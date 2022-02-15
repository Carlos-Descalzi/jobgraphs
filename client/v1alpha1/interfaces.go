package v1alpha1

import (
	"context"
	"time"

	"net/http"

	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type JobGraphInterface interface {
	Create(ctx context.Context, jobGraph *types.JobGraph) (*types.JobGraph, error)
	Update(ctx context.Context, jobGraph *types.JobGraph) (*types.JobGraph, error)
	Delete(ctx context.Context, namespace string, name string, options metav1.DeleteOptions) error
	Get(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*types.JobGraph, error)
	List(ctx context.Context, namespace string, options metav1.ListOptions) (*types.JobGraphList, error)
	Watch(ctx context.Context, namespace string, options metav1.ListOptions) (watch.Interface, error)
}

type jobGraphInterfaceImpl struct {
	client rest.Interface
}

func JobGraphInterfaceNew(config *rest.Config, httpClient *http.Client) (JobGraphInterface, error) {
	cfgCopy := *config

	gv := types.SchemeGroupVersion
	cfgCopy.GroupVersion = &gv
	cfgCopy.APIPath = "/apis"
	cfgCopy.NegotiatedSerializer = Codecs

	if cfgCopy.UserAgent == "" {
		cfgCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	restClient, err := rest.RESTClientForConfigAndClient(&cfgCopy, httpClient)

	if err != nil {
		return nil, err
	}

	return &jobGraphInterfaceImpl{client: restClient}, nil
}

func (j *jobGraphInterfaceImpl) Create(ctx context.Context, jobGraph *types.JobGraph) (*types.JobGraph, error) {
	result := &types.JobGraph{}
	err := j.client.Post().
		Namespace(jobGraph.Namespace).
		Resource(types.ResourceType).
		Body(jobGraph).
		Do(ctx).
		Into(result)
	return result, err
}

func (j *jobGraphInterfaceImpl) Update(ctx context.Context, jobGraph *types.JobGraph) (*types.JobGraph, error) {
	result := &types.JobGraph{}
	err := j.client.Put().
		Namespace(jobGraph.Namespace).
		Resource(types.ResourceType).
		Name(jobGraph.Name).
		Body(jobGraph).
		Do(ctx).
		Into(result)
	return result, err
}
func (j *jobGraphInterfaceImpl) Delete(ctx context.Context, namespace string, name string, options metav1.DeleteOptions) error {
	return j.client.Delete().
		Namespace(namespace).
		Resource(types.ResourceType).
		Name(name).
		Body(options).
		Do(ctx).
		Error()
}
func (j *jobGraphInterfaceImpl) Get(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*types.JobGraph, error) {

	result := &types.JobGraph{}

	err := j.client.Get().
		Resource(types.ResourceType).
		Namespace(namespace).
		Name(name).
		VersionedParams(&options, ParameterCodec).
		Do(ctx).
		Into(result)

	return result, err
}
func (j *jobGraphInterfaceImpl) List(ctx context.Context, namespace string, options metav1.ListOptions) (*types.JobGraphList, error) {

	var timeout time.Duration
	if options.TimeoutSeconds != nil {
		timeout = time.Duration(*options.TimeoutSeconds) * time.Second
	}

	result := &types.JobGraphList{}
	err := j.client.Get().
		Resource(types.ResourceType).
		Namespace(namespace).
		VersionedParams(&options, ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)

	return result, err
}
func (j *jobGraphInterfaceImpl) Watch(ctx context.Context, namespace string, options metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if options.TimeoutSeconds != nil {
		timeout = time.Duration(*options.TimeoutSeconds) * time.Second
	}
	options.Watch = true

	return j.client.Get().
		Namespace(namespace).
		Resource(types.ResourceType).
		VersionedParams(&options, ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
