package common

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// MetadataNamespace is the namespace for service mesh metadata (labels, annotations)
	MetadataNamespace = "maistra.io"

	// CreatedByKey is used in annotations to mark ServiceMeshMemberRolls created by the ServiceMeshMember controller
	CreatedByKey = MetadataNamespace + "/created-by"

	// OwnerKey represents the mesh (namespace) to which the resource relates
	OwnerKey = MetadataNamespace + "/owner"

	// MemberOfKey represents the mesh (namespace) to which the resource relates
	MemberOfKey = MetadataNamespace + "/member-of"

	// GenerationKey represents the generation to which the resource was last reconciled
	GenerationKey = MetadataNamespace + "/generation"

	// MeshGenerationKey represents the generation of the service mesh to which the resource was last reconciled
	MeshGenerationKey = MetadataNamespace + "/mesh-generation"

	// InternalKey is used to identify the resource as being internal to the mesh itself (i.e. should not be applied to members)
	InternalKey = MetadataNamespace + "/internal"

	// FinalizerName is the finalizer name the controllers add to any resources that need to be finalized during deletion
	FinalizerName = MetadataNamespace + "/istio-operator"

	// KubernetesAppNamespace is the common namespace for application information
	KubernetesAppNamespace    = "app.kubernetes.io"
	KubernetesAppNameKey      = KubernetesAppNamespace + "/name"
	KubernetesAppInstanceKey  = KubernetesAppNamespace + "/instance"
	KubernetesAppVersionKey   = KubernetesAppNamespace + "/version"
	KubernetesAppComponentKey = KubernetesAppNamespace + "/component"
	KubernetesAppPartOfKey    = KubernetesAppNamespace + "/part-of"
	KubernetesAppManagedByKey = KubernetesAppNamespace + "/managed-by"

	// MemberRollName is the only name we allow for ServiceMeshMemberRoll objects
	MemberRollName = "default"

	// MemberName is the only name we allow for ServiceMeshMember objects
	MemberName = "default"
)

func (rm *ResourceManager) FetchOwnedResources(gvk schema.GroupVersionKind, owner, namespace string) ([]runtime.Object, error) {
	labelSelector := map[string]string{OwnerKey: owner}
	return rm.fetchResources(gvk, client.MatchingLabels(labelSelector).InNamespace(namespace))
}

func (rm *ResourceManager) FetchMeshResources(gvk schema.GroupVersionKind, mesh, namespace string) ([]runtime.Object, error) {
	labelSelector := map[string]string{MemberOfKey: mesh}
	return rm.fetchResources(gvk, client.MatchingLabels(labelSelector).InNamespace(namespace))
}

func (rm *ResourceManager) fetchResources(gvk schema.GroupVersionKind, opts *client.ListOptions) ([]runtime.Object, error) {
	if !strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = gvk.Kind + "List"
	}
	list, err := rm.Scheme.New(gvk)
	if err != nil {
		ulist := &unstructured.UnstructuredList{}
		ulist.SetGroupVersionKind(gvk)
		list = ulist
	}
	err = rm.Client.List(context.TODO(), opts, list)
	if err != nil {
		return nil, err
	}
	return meta.ExtractList(list)
}
