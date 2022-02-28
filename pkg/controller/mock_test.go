package controller

import (
	"context"
	"fmt"

	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type MockWatcher struct {
	channel chan watch.Event
}

func (m MockWatcher) Stop() {
	close(m.channel)
}

func (m MockWatcher) ResultChan() <-chan watch.Event {
	return m.channel
}

type watchJobsCall struct {
	context context.Context
	options metav1.ListOptions
}

type createJobCall struct {
	context   context.Context
	namespace string
	job       *v1.Job
}

type listJobsCall struct {
	context     context.Context
	namespace   string
	listOptions metav1.ListOptions
}

type mockJobHandler struct {
	jobList          *v1.JobList
	watchIfce        watch.Interface
	error            error
	createJobChannel chan createJobCall
	watchJobsCall    *watchJobsCall
	//createJobCall *createJobCall
	listJobsCall *listJobsCall
}

func (h *mockJobHandler) WatchJobs(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
	h.watchJobsCall = &watchJobsCall{context: ctx, options: options}
	return h.watchIfce, h.error
}

func (h *mockJobHandler) CreateJob(ctx context.Context, namespace string, job *v1.Job) error {
	h.createJobChannel <- createJobCall{context: ctx, namespace: namespace, job: job}
	return h.error
}

func (h *mockJobHandler) ListJobs(ctx context.Context, namespace string, options metav1.ListOptions) (*v1.JobList, error) {
	h.listJobsCall = &listJobsCall{context: ctx, namespace: namespace, listOptions: options}
	return h.jobList, h.error
}

type updateJobGraphCall struct {
	context  context.Context
	jobGraph *types.JobGraph
}

type mockJobGrapHandler struct {
	watchIfce         watch.Interface
	error             error
	jobGraphList      *types.JobGraphList
	jobGraph          *types.JobGraph
	updateCallChannel chan updateJobGraphCall
	updateCall        *updateJobGraphCall
}

func (h *mockJobGrapHandler) Create(ctx context.Context, jobGraph *types.JobGraph) (*types.JobGraph, error) {
	return jobGraph, h.error
}

func (h *mockJobGrapHandler) Update(ctx context.Context, jobGraph *types.JobGraph) (*types.JobGraph, error) {
	if h.updateCallChannel != nil {
		h.updateCallChannel <- updateJobGraphCall{ctx, jobGraph}
	} else {
		h.updateCall = &updateJobGraphCall{ctx, jobGraph}
	}
	return jobGraph, h.error
}
func (h *mockJobGrapHandler) Delete(ctx context.Context, namespace string, name string, options metav1.DeleteOptions) error {
	return h.error
}
func (h *mockJobGrapHandler) Get(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*types.JobGraph, error) {
	fmt.Printf("Get Job Graph %s/%s", namespace, name)
	return h.jobGraph, h.error
}
func (h *mockJobGrapHandler) List(ctx context.Context, namespace string, options metav1.ListOptions) (*types.JobGraphList, error) {
	return h.jobGraphList, h.error
}
func (h *mockJobGrapHandler) Watch(ctx context.Context, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	return h.watchIfce, h.error
}

func makeLogger() *zap.SugaredLogger {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	zapEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	zapConfig := zap.Config{
		Level: level,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "console",
		EncoderConfig:    zapEncoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil
	}
	return logger.Sugar()
}
