package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	types "ced.io/jobgraphs/pkg/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	Scheme             = runtime.NewScheme()
	Codecs             = serializer.NewCodecFactory(Scheme)
	ParameterCodec     = runtime.NewParameterCodec(Scheme)
	localSchemeBuilder = runtime.SchemeBuilder{
		types.AddToScheme,
	}
	AddToScheme = localSchemeBuilder.AddToScheme
)

func init() {
	metav1.AddToGroupVersion(Scheme, types.SchemeGroupVersion)
	utilruntime.Must(AddToScheme(Scheme))

}
