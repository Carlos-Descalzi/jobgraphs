package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	ifces "ced.io/jobgraphs/client/v1alpha1"
	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
	graphs "ced.io/jobgraphs/pkg/util"
	"go.uber.org/zap"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
)

type JobInfo struct {
	graph   *types.JobGraph
	jobName string
}

type LockInfo struct {
	workerNumber int
	w            *sync.WaitGroup
}

type JobHandler interface {
	WatchJobs(context.Context, metav1.ListOptions) (watch.Interface, error)
	CreateJob(context.Context, string, *v1.Job) error
	ListJobs(context.Context, string, metav1.ListOptions) (*v1.JobList, error)
}

type DefaultJobHandler struct {
	kubeClient *clientset.Clientset
}

func DefaultJobHandlerNew(kubeClient *clientset.Clientset) *DefaultJobHandler {
	return &DefaultJobHandler{kubeClient: kubeClient}
}

func (h *DefaultJobHandler) WatchJobs(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
	return h.kubeClient.
		BatchV1().
		Jobs(metav1.NamespaceAll).
		Watch(
			ctx,
			options,
		)
}

func (h *DefaultJobHandler) CreateJob(ctx context.Context, namespace string, job *v1.Job) error {
	_, err := h.kubeClient.BatchV1().
		Jobs(namespace).
		Create(ctx, job, metav1.CreateOptions{})
	return err
}

func (h *DefaultJobHandler) ListJobs(ctx context.Context, namespace string, options metav1.ListOptions) (*v1.JobList, error) {
	return h.kubeClient.
		BatchV1().
		Jobs(namespace).
		List(
			ctx,
			options,
		)
}

type JobGraphsController struct {
	active            bool
	inputWorkerCount  int
	outputWorkerCount int
	jobHandler        JobHandler
	jobGraphIfce      ifces.JobGraphInterface
	jobWatcher        watch.Interface
	jobGraphWatcher   watch.Interface
	ctx               context.Context
	dispatchQueue     chan JobInfo
	inputQueue        chan *v1.Job
	logger            *zap.SugaredLogger
	graphsInProcess   map[string]LockInfo
	inProcessMutex    sync.RWMutex
	//nodesInProcess    map[string]bool
	//graphCheckMutex   sync.RWMutex
}

func JobGraphsControllerNew(
	inputWorkerCount int,
	outputWorkerCount int,
	jobHandler JobHandler,
	jobGraphIfce ifces.JobGraphInterface,
	logger *zap.SugaredLogger) (*JobGraphsController, error) {

	if inputWorkerCount <= 0 {
		return nil, fmt.Errorf("inputWorkerCount must be > 0")
	}

	if outputWorkerCount <= 0 {
		return nil, fmt.Errorf("outputWorkerCount must be > 0")
	}

	if jobHandler == nil {
		return nil, fmt.Errorf("jobHandler client is nil")
	}
	if jobGraphIfce == nil {
		return nil, fmt.Errorf("Job graph interface is nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("Logger is nil")
	}

	controller := JobGraphsController{
		jobHandler:        jobHandler,
		inputWorkerCount:  inputWorkerCount,
		outputWorkerCount: outputWorkerCount,
		jobGraphIfce:      jobGraphIfce,
		ctx:               context.Background(),
		dispatchQueue:     make(chan JobInfo),
		inputQueue:        make(chan *v1.Job),
		logger:            logger,
		graphsInProcess:   make(map[string]LockInfo),
		//nodesInProcess:    make(map[string]bool),
	}

	return &controller, nil
}

func (c *JobGraphsController) Start() {
	c.active = true
	go c.watchGraphs()
	go c.watchJobs()
	for i := 0; i < c.inputWorkerCount; i++ {
		go c.processIncomingJobs(i)
	}
	for i := 0; i < c.outputWorkerCount; i++ {
		go c.dispatchJobs(i)
	}
}

func (c *JobGraphsController) Stop() {
	c.active = false
	close(c.inputQueue)
	close(c.dispatchQueue)
	if c.jobWatcher != nil {
		c.jobWatcher.Stop()
	}
	if c.jobGraphWatcher != nil {
		c.jobGraphWatcher.Stop()
	}
}

func (c *JobGraphsController) watchGraphs() {

	for c.active {
		watcher, err := c.jobGraphIfce.Watch(c.ctx, metav1.NamespaceAll, metav1.ListOptions{})

		if err == nil {
			c.jobGraphWatcher = watcher
			channel := c.jobGraphWatcher.ResultChan()

			for c.active {
				evt, ok := <-channel

				if ok {
					switch evt.Type {
					case watch.Added:
						c.startJobGraph(evt.Object.(*types.JobGraph))
					case watch.Error:
						c.logger.Errorf("Error with job graph: %s", evt.Object.(*metav1.Status))
					}
				} else {
					if c.active {
						// if there is a deserialization issue the channel will break,
						// so exit from here and get a new watcher after a while.
						time.Sleep(1 * time.Second)
					} // otherwise it means the channel has been closed.
					break
				}
			}

		} else {
			c.logger.Error("Unable to watch for job graphs", err)
			time.Sleep(1 * time.Second)
		}
	}
}

func (c *JobGraphsController) watchJobs() {

	watcher, err := c.jobHandler.WatchJobs(
		c.ctx,
		metav1.ListOptions{
			LabelSelector: "graph,node",
		},
	)

	if err == nil {
		c.jobWatcher = watcher
		channel := c.jobWatcher.ResultChan()

		for c.active {
			evt, ok := <-channel

			if ok {
				switch evt.Type {

				case watch.Modified:
					c.checkJob(evt.Object.(*v1.Job))
				}
			}
		}
	}
}

func (c *JobGraphsController) startJobGraph(jobGraph *types.JobGraph) {
	if jobGraph.Status.State != types.Success &&
		jobGraph.Status.State != types.Error {

		err := graphs.CheckAcyclic(&jobGraph.Spec.Graph)

		if err != nil {
			c.logger.Errorf("Unable to run job graph %s/%s: %s", jobGraph.Namespace, jobGraph.Name, err)
			c.setGraphError(jobGraph, err)
		} else {
			c.logger.Infof("Starting job graph %s/%s", jobGraph.Namespace, jobGraph.Name)
			c.setGraphStarted(jobGraph)

			roots := graphs.Roots(&jobGraph.Spec.Graph)

			for i := 0; i < len(roots); i++ {
				c.dispatchQueue <- JobInfo{graph: jobGraph, jobName: roots[i]}
			}
		}

	}
}

func (c *JobGraphsController) checkJob(job *v1.Job) {
	if job.Status.CompletionTime != nil {
		c.logger.Infof(
			"Job %s/%s finished, checking graph %s - node %s",
			job.Namespace,
			job.Name,
			job.Labels["graph"],
			job.Labels["node"],
		)
		c.inputQueue <- job
	}
}

func (*JobGraphsController) isJobSuccess(job *v1.Job) bool {
	var completions int32 = 1

	if job.Spec.Completions != nil {
		completions = *job.Spec.Completions
	}

	return completions == job.Status.Succeeded
}

func (c *JobGraphsController) dispatchJobs(workerNumber int) {
	for c.active {
		next, ok := <-c.dispatchQueue

		if ok {
			//if c.lockNode(next.graph.Namespace, next.graph.Name, next.jobName) {
			c.logger.Infof("Worker #%d: Dispatching job %s/%s", workerNumber, next.graph.Namespace, next.jobName)

			if !c.jobForNodeExists(next.graph, next.jobName) {
				job := c.makeJob(next)

				if job != nil {
					err := c.jobHandler.CreateJob(c.ctx, next.graph.Namespace, job)

					if err != nil {
						c.logger.Error(err)
					}
				}
			} else {
				c.logger.Infof("Worker #%d: Job for node %s/%s/%s already created", workerNumber, next.graph.Namespace, next.graph.Name, next.jobName)
			}
			//c.unlockNode(next.graph.Namespace, next.graph.Name, next.jobName)
			//} else {
			//	c.logger.Info("Worker #%d: Node %s already taken by other worker, skipping", workerNumber, next.jobName)
			//}
		}
	}
}

func (c *JobGraphsController) makeJob(jobInfo JobInfo) *v1.Job {

	templates := jobInfo.graph.Spec.JobTemplates

	for i := 0; i < len(templates); i++ {
		name := templates[i].Name
		if name == jobInfo.jobName {
			job := v1.Job{Spec: *templates[i].Spec.DeepCopy()}
			job.Name = fmt.Sprintf("%s-%v", name, time.Now().Unix())
			job.Namespace = jobInfo.graph.Namespace
			job.Labels = make(map[string]string)
			job.Labels["graph-id"] = string(jobInfo.graph.UID)
			job.Labels["graph"] = jobInfo.graph.Name
			job.Labels["node"] = jobInfo.jobName
			for j := 0; j < len(job.Spec.Template.Spec.Containers); j++ {
				job.Spec.Template.Spec.Containers[j].Name = fmt.Sprintf("%s-%d", job.Name, j)
			}
			return &job
		}
	}
	c.logger.Error("Job definition not found")

	return nil
}

func (c *JobGraphsController) fetchGraph(namespace string, name string) (*types.JobGraph, error) {
	return c.jobGraphIfce.Get(c.ctx, namespace, name, metav1.GetOptions{})
}

func (c *JobGraphsController) processIncomingJobs(workerNumber int) {
	for c.active {
		job, ok := <-c.inputQueue

		if ok {
			graphName := job.Labels["graph"]
			nodeName := job.Labels["node"]

			c.logger.Infof("Processing outcomes for %s/%s:%s", job.Namespace, graphName, nodeName)

			graph, err := c.fetchGraph(job.Namespace, graphName)
			if err == nil {
				c.lockGraph(graph, func() {
					if c.isJobSuccess(job) {
						graph.Status.SucceededJobs = append(graph.Status.SucceededJobs, job.Name)
					} else {
						graph.Status.FailedJobs = append(graph.Status.FailedJobs, job.Name)
					}

					nodeCount := graphs.NodeCount(&graph.Spec.Graph)
					jobCount := len(graph.Status.SucceededJobs) + len(graph.Status.FailedJobs)

					nextNodes := c.nextNodes(graph, nodeName)

					canContinue := c.canContinue(graph)

					c.logger.Debugf("Graph %s/%s: next node count:%d, can continue: %v\n",
						graph.Namespace,
						graph.Name,
						len(nextNodes),
						canContinue,
					)
					if !canContinue || jobCount == nodeCount {
						c.logger.Infof("Graph %s/%s finished", job.Namespace, graphName)
						c.setGraphFinished(graph)
					} else {
						for i := 0; i < len(nextNodes); i++ {
							c.logger.Debugf("Next node for graph %s/%s:%s", job.Namespace, graphName, nextNodes[i])
							c.dispatchQueue <- JobInfo{graph: graph, jobName: nextNodes[i]}
						}
						c.updateGraph(graph)
					}
				})
			}
			c.logger.Infof("Job check finished %s/%s:%s", job.Namespace, graphName, nodeName)
		} else {
			c.logger.Info("No more jobs to check")
		}
	}
}

func (c *JobGraphsController) canContinue(graph *types.JobGraph) bool {
	return (graph.Spec.JobFailureCondition == "" ||
		graph.Spec.JobFailureCondition == types.Continue) ||
		len(graph.Status.FailedJobs) == 0

}

func (c *JobGraphsController) nextNodes(graph *types.JobGraph, nodeName string) []string {
	//c.graphCheckMutex.Lock()
	//defer c.graphCheckMutex.Unlock()
	nextNodes := graphs.Outgoing(&graph.Spec.Graph, nodeName)

	runningJobs := c.getRunningNodes(graph)

	var result []string

	for i := 0; i < len(nextNodes); i++ {
		otherIncoming := graphs.Incoming(&graph.Spec.Graph, nextNodes[i])

		if !c.intersect(runningJobs, otherIncoming) {
			result = append(result, nextNodes[i])
		}

	}

	return result
}

func (c *JobGraphsController) intersect(array1 []string, array2 []string) bool {
	m := make(map[string]bool)

	for i := 0; i < len(array1); i++ {
		m[array1[i]] = true
	}

	for i := 0; i < len(array2); i++ {
		if _, ok := m[array2[i]]; ok {
			return true
		}
	}
	return false
}

func (c *JobGraphsController) getRunningNodes(graph *types.JobGraph) []string {

	var result []string

	jobs, err := c.jobHandler.ListJobs(
		c.ctx,
		graph.Namespace,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("graph-id=%s", string(graph.UID)),
		},
	)

	if err == nil {
		for i := 0; i < len(jobs.Items); i++ {
			if c.isJobRunning(&jobs.Items[i]) {
				result = append(result, jobs.Items[i].Labels["node"])
			}
		}
	} else {
		c.logger.Error("Error fetching jobs.", err)
	}

	return result
}

func (c *JobGraphsController) jobForNodeExists(graph *types.JobGraph, nodeName string) bool {
	jobs, err := c.jobHandler.ListJobs(
		c.ctx,
		graph.Namespace,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("graph-id=%s, node=%s", string(graph.UID), nodeName),
		},
	)
	if err == nil {
		return len(jobs.Items) != 0
	}
	return false
}

func (c *JobGraphsController) isJobRunning(job *v1.Job) bool {

	var completions int32 = 1

	if job.Spec.Completions != nil {
		completions = *job.Spec.Completions
	}

	var total = job.Status.Succeeded + job.Status.Failed

	return completions > total
}

func (c *JobGraphsController) setGraphStarted(graph *types.JobGraph) {
	now := metav1.Now()
	graph.Status.StartTime = &now

	c.updateState(graph, types.InProgress)
}

func (c *JobGraphsController) setGraphError(graph *types.JobGraph, err error) {
	graph.Status.Reason = err.Error()
	c.updateState(graph, types.Error)
}

func (c *JobGraphsController) setGraphFinished(graph *types.JobGraph) {

	var state types.JobGraphState

	if len(graph.Status.FailedJobs) > 0 {
		c.logger.Debugf("Job graph %s has failed jobs %d", graph.Name, graph.Status.FailedJobs)
		state = types.Error
	} else {
		state = types.Success
	}
	now := metav1.Now()
	graph.Status.EndTime = &now

	c.updateState(graph, state)
}

func (c *JobGraphsController) updateState(graph *types.JobGraph, state types.JobGraphState) {
	graph.Status.State = state
	c.logger.Infof("Set graph state %s/%s: %s", graph.Namespace, graph.Name, state)
	c.updateGraph(graph)
}

func (c *JobGraphsController) updateGraph(graph *types.JobGraph) {
	c.jobGraphIfce.Update(c.ctx, graph)
}

/*
func (c *JobGraphsController) lockNode(namespace string, graph string, node string) bool {
	c.graphCheckMutex.Lock()
	defer c.graphCheckMutex.Unlock()

	key := fmt.Sprintf("%s-%s-%s", namespace, graph, node)

	_, ok := c.nodesInProcess[key]

	if ok {
		return false
	}
	c.nodesInProcess[key] = true
	return true
}

func (c *JobGraphsController) unlockNode(namespace string, graph string, node string) {
	c.graphCheckMutex.Lock()
	defer c.graphCheckMutex.Unlock()

	key := fmt.Sprintf("%s-%s-%s", namespace, graph, node)

	_, ok := c.nodesInProcess[key]

	if ok {
		delete(c.nodesInProcess, key)
	}
}
*/
/**
  Avoid 2 workers do updates on the same graph at the same time
**/
func (c *JobGraphsController) lockGraph(graph *types.JobGraph, function func()) {

	key := fmt.Sprintf("%s-%s", graph.Namespace, graph.Name)

	c.inProcessMutex.Lock()
	lock, ok := c.graphsInProcess[key]
	c.inProcessMutex.Unlock()

	if ok {
		lock.w.Wait()
	} else {
		c.inProcessMutex.Lock()
		c.graphsInProcess[key] = LockInfo{w: &sync.WaitGroup{}, workerNumber: 0}
		c.graphsInProcess[key].w.Add(1)
		c.inProcessMutex.Unlock()
	}

	function()

	c.inProcessMutex.Lock()
	delete(c.graphsInProcess, key)
	c.inProcessMutex.Unlock()
}
