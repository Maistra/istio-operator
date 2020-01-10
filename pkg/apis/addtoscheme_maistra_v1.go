package apis

import (
	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"

	networkv1 "github.com/openshift/api/network/v1"
	routev1 "github.com/openshift/api/route/v1"
	k8score "k8s.io/client-go/kubernetes/scheme"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1.SchemeBuilder.AddToScheme,
		networkv1.AddToScheme,
		routev1.AddToScheme,
		k8score.AddToScheme)
}
