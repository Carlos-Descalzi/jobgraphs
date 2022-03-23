package controller

import (
	"fmt"
	"testing"
	"time"

	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestJobGraphsControllerNew(t *testing.T) {

	_, err := JobGraphsControllerNew(0, 1, nil, nil, nil)

	if err == nil {
		t.Error()
	}
	_, err = JobGraphsControllerNew(1, 0, nil, nil, nil)

	if err == nil {
		t.Error()
	}
	_, err = JobGraphsControllerNew(1, 1, nil, nil, nil)

	if err == nil {
		t.Error()
	}

	jobHandler := &mockJobHandler{}
	jobHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}

	_, err = JobGraphsControllerNew(1, 1, jobHandler, nil, nil)

	if err == nil {
		t.Error()
	}

	jobGraphHandler := &mockJobGrapHandler{}
	jobGraphHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}

	_, err = JobGraphsControllerNew(1, 1, jobHandler, jobGraphHandler, nil)

	if err == nil {
		t.Error()
	}

	logger := makeLogger()

	controller, err := JobGraphsControllerNew(1, 1, jobHandler, jobGraphHandler, logger)

	if err != nil {
		t.Error()
	}

	if controller == nil {
		t.Error()
	}

	if controller.active {
		t.Error()
	}

	controller.Start()

	if !controller.active {
		t.Error()
	}

	controller.Stop()

	if controller.active {
		t.Error()
	}
}

func TestIsJobSuccess(t *testing.T) {
	controller := JobGraphsController{}

	job := v1.Job{}
	job.Spec = v1.JobSpec{}

	if controller.isJobSuccess(&job) {
		t.Error()
	}

	job.Status.Succeeded = 1

	if !controller.isJobSuccess(&job) {
		t.Error()
	}

}

func TestIsJobRunning(t *testing.T) {
	controller := JobGraphsController{}

	job := v1.Job{}

	if !controller.isJobRunning(&job) {
		t.Error()
	}

	job.Status.Succeeded = 1

	if controller.isJobRunning(&job) {
		t.Error()
	}

	var c int32 = 3

	job.Spec.Completions = &c

	if !controller.isJobRunning(&job) {
		t.Error()
	}

	job.Status.Failed = 2

	if controller.isJobRunning(&job) {
		t.Error()
	}
}

func TestCanContinue(t *testing.T) {
	controller := JobGraphsController{}

	graph := types.JobGraph{}

	if !controller.canContinue(&graph) {
		t.Error()
	}

	graph.Status.FailedJobs = append(graph.Status.FailedJobs, "job1")

	if !controller.canContinue(&graph) {
		t.Error()
	}

	graph.Spec.JobFailureCondition = types.Fail

	if controller.canContinue(&graph) {
		t.Error()
	}

	graph.Status.FailedJobs = []string{}

	if !controller.canContinue(&graph) {
		t.Error()
	}
}

func TestIntersect(t *testing.T) {
	controller := JobGraphsController{}

	if !controller.intersect([]string{"str1", "str2"}, []string{"str1", "str3"}) {
		t.Error()
	}
	if controller.intersect([]string{"str1", "str2"}, []string{"str4", "str3"}) {
		t.Error()
	}
	if controller.intersect([]string{"str1", "str2"}, []string{}) {
		t.Error()
	}
	if controller.intersect([]string{}, []string{"str1", "str2"}) {
		t.Error()
	}
}

func TestDispatchJobs(t *testing.T) {

	jobHandler := &mockJobHandler{}
	jobHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	jobHandler.createJobChannel = make(chan createJobCall)
	jobGraphHandler := &mockJobGrapHandler{}
	jobGraphHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	logger := makeLogger()

	controller, err := JobGraphsControllerNew(1, 1, jobHandler, jobGraphHandler, logger)
	if err != nil {
		t.Error()
	}
	jobHandler.jobList = &v1.JobList{Items: []v1.Job{}}

	controller.active = true

	go controller.dispatchJobs(0)

	graph := types.JobGraph{}
	graph.Spec.JobTemplates = append(
		graph.Spec.JobTemplates, v1.JobTemplateSpec{},
	)
	graph.Spec.JobTemplates[0].Name = "job1"
	controller.dispatchQueue <- JobInfo{jobName: "job1", graph: &graph}

	stop := func() {
		controller.active = false
		jobHandler.watchIfce.Stop()
		jobGraphHandler.watchIfce.Stop()
	}

	select {
	case <-jobHandler.createJobChannel:
		stop()
		// ok
	case <-time.After(2 * time.Millisecond):
		stop()
		t.Error()
	}

}

func TestProcessIncomingJobs(t *testing.T) {
	jobHandler := &mockJobHandler{}
	jobHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	jobHandler.createJobChannel = make(chan createJobCall)
	jobGraphHandler := &mockJobGrapHandler{}
	jobGraphHandler.updateCallChannel = make(chan updateJobGraphCall)
	jobGraphHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	jobGraphHandler.jobGraph = &types.JobGraph{}
	jobGraphHandler.jobGraph.Name = "graph1"
	jobGraphHandler.jobGraph.Spec.Graph.Edges = []types.Edge{{Source: "node1", Target: "node2"}}
	jobHandler.jobList = &v1.JobList{Items: []v1.Job{{}}}
	jobHandler.jobList.Items[0].Labels = make(map[string]string)
	jobHandler.jobList.Items[0].Labels["graph"] = "graph1"
	logger := makeLogger()

	controller, err := JobGraphsControllerNew(1, 1, jobHandler, jobGraphHandler, logger)
	if err != nil {
		t.Error()
	}

	controller.active = true
	go controller.processIncomingJobs(0)

	job := v1.Job{}
	job.Labels = make(map[string]string)
	job.Labels["graph"] = "graph1"
	job.Labels["node"] = "node1"

	controller.inputQueue <- &job

	stop := func() {
		controller.active = false
		jobHandler.watchIfce.Stop()
		jobGraphHandler.watchIfce.Stop()
	}

	select {
	case job := <-controller.dispatchQueue:
		stop()
		fmt.Printf("Job received\n")
		if job.graph == nil {
			t.Error()
		}
		if len(job.jobName) == 0 {
			t.Error()
		}

	case <-time.After(2 * time.Millisecond):
		stop()
		t.Error()
	}
}

func TestSetGraphFinished(t *testing.T) {

	graph := types.JobGraph{}
	graph.Status.FailedJobs = []string{"job1"}

	jobHandler := &mockJobHandler{}
	jobHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	jobHandler.createJobChannel = make(chan createJobCall)
	jobGraphHandler := &mockJobGrapHandler{}
	jobGraphHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	logger := makeLogger()

	controller, err := JobGraphsControllerNew(1, 1, jobHandler, jobGraphHandler, logger)

	if err != nil {
		t.Error()
	}

	controller.setGraphFinished(&graph)

	if jobGraphHandler.updateCall == nil {
		t.Error()
	}

	if graph.Status.State != types.Error {
		t.Error()
	}

	if graph.Status.EndTime == nil {
		t.Error()
	}

	jobGraphHandler.updateCall = nil

	graph.Status.EndTime = nil
	graph.Status.FailedJobs = []string{}
	graph.Status.SucceededJobs = []string{"job1"}

	controller.setGraphFinished(&graph)

	if jobGraphHandler.updateCall == nil {
		t.Error()
	}

	if graph.Status.State == types.Error {
		t.Error()
	}

	if graph.Status.EndTime == nil {
		t.Error()
	}
}
func TestSetGraphStarted(t *testing.T) {

	graph := types.JobGraph{}
	graph.Status.FailedJobs = []string{"job1"}

	jobHandler := &mockJobHandler{}
	jobHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	jobHandler.createJobChannel = make(chan createJobCall)
	jobGraphHandler := &mockJobGrapHandler{}
	jobGraphHandler.watchIfce = &MockWatcher{channel: make(chan watch.Event, 1)}
	logger := makeLogger()

	controller, err := JobGraphsControllerNew(1, 1, jobHandler, jobGraphHandler, logger)

	if err != nil {
		t.Error()
	}

	controller.setGraphStarted(&graph)

	if graph.Status.StartTime == nil {
		t.Error()
	}

	if jobGraphHandler.updateCall == nil {
		t.Error()
	}

}
