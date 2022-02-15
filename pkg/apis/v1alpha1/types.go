package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type objectKind struct {
	kind schema.GroupVersionKind
}

func (o *objectKind) SetGroupVersionKind(kind schema.GroupVersionKind) {
	o.kind = kind
}

func (o objectKind) GroupVersionKind() schema.GroupVersionKind {
	return o.kind
}

func newObjectKind(groupVersion schema.GroupVersion, kind string) schema.ObjectKind {

	return &objectKind{
		kind: schema.GroupVersionKind{
			Group:   groupVersion.Group,
			Version: groupVersion.Version,
			Kind:    kind,
		},
	}
}

const ResourceType string = "jobgraphs"

var kind = newObjectKind(SchemeGroupVersion, "JobGraph")

var listKind = newObjectKind(SchemeGroupVersion, "JobGraphList")

type JobFailureCondition string

const (
	Continue JobFailureCondition = "continue"
	Fail     JobFailureCondition = "fail"
)

type JobGraph struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec JobGraphSpec `json:"spec"`

	Status JobGraphStatus `json:"status"`
}

func (j JobGraph) GetObjectKind() schema.ObjectKind {
	return kind
}
func (j JobGraph) DeepCopyObject() runtime.Object {
	return nil
}

type JobGraphSpec struct {
	Graph GraphSpec `json:"graph"`

	JobFailureCondition JobFailureCondition `json:"jobFailureCondition"`

	JobTemplates []batchv1.JobTemplateSpec `json:"jobTemplates"`
}

type GraphSpec struct {
	Edges []Edge `json:"edges"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type JobGraphState string

const (
	InProgress JobGraphState = "inProgress"
	Success    JobGraphState = "success"
	Error      JobGraphState = "error"
)

type JobGraphStatus struct {
	State JobGraphState `json:"state"`

	Reason string `json:"reason"`

	Active []v1.ObjectReference `json:"active"`

	SucceededJobs []string `json:"succeededJobs"`

	FailedJobs []string `json:"failedJobs"`

	StartTime *metav1.Time `json:"startTime"`

	EndTime *metav1.Time `json:"endTime"`
}

type JobGraphList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []JobGraph `json:"items"`
}

func (j JobGraphList) GetObjectKind() schema.ObjectKind {
	return listKind
}
func (j JobGraphList) DeepCopyObject() runtime.Object {
	return nil
}
