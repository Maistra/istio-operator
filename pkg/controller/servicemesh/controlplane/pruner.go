package controlplane

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/maistra/istio-operator/pkg/controller/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// XXX: move this into a ConfigMap so users can override things if they add new types in customized charts
	// ordered by which types should be deleted, first to last
	namespacedResources = []schema.GroupVersionKind{
		schema.GroupVersionKind{Group: "kiali.io", Version: "v1alpha1", Kind: "Kiali"},
		schema.GroupVersionKind{Group: "autoscaling", Version: "v2beta1", Kind: "HorizontalPodAutoscaler"},
		schema.GroupVersionKind{Group: "policy", Version: "v1beta1", Kind: "PodDisruptionBudget"},
		schema.GroupVersionKind{Group: "route.openshift.io", Version: "v1", Kind: "Route"},
		schema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: "Deployment"},
		schema.GroupVersionKind{Group: "apps", Version: "v1beta1", Kind: "StatefulSet"},
		schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
		schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "DaemonSet"},
		schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "Deployment"},
		schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "Ingress"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Endpoints"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
		schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1beta1", Kind: "RoleBinding"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1beta1", Kind: "Role"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
		schema.GroupVersionKind{Group: "authentication.istio.io", Version: "v1alpha1", Kind: "Policy"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "adapter"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "attributemanifest"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "handler"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "kubernetes"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "logentry"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "metric"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "rule"},
		schema.GroupVersionKind{Group: "config.istio.io", Version: "v1alpha2", Kind: "template"},
		schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1alpha3", Kind: "DestinationRule"},
		schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1alpha3", Kind: "EnvoyFilter"},
		schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1alpha3", Kind: "Gateway"},
		schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1alpha3", Kind: "VirtualService"},
		schema.GroupVersionKind{Group: "jaegertracing.io", Version: "v1", Kind: "jaeger"},
		schema.GroupVersionKind{Group: "authentication.maistra.io", Version: "v1", Kind: "ServiceMeshPolicy"},
	}

	// ordered by which types should be deleted, first to last
	nonNamespacedResources = []schema.GroupVersionKind{
		schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1beta1", Kind: "MutatingWebhookConfiguration"},
		schema.GroupVersionKind{Group: "admissionregistration.k8s.io", Version: "v1beta1", Kind: "ValidatingWebhookConfiguration"},
		schema.GroupVersionKind{Group: "certmanager.k8s.io", Version: "v1alpha1", Kind: "ClusterIssuer"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
	}
)

func (r *ControlPlaneReconciler) prune(generation string) error {
	allErrors := []error{}
	err := r.pruneResources(namespacedResources, generation, r.Instance.Namespace)
	if err != nil {
		allErrors = append(allErrors, err)
	}
	err = r.pruneResources(namespacedResources, generation, r.OperatorNamespace)
	if err != nil {
		allErrors = append(allErrors, err)
	}
	err = r.pruneResources(nonNamespacedResources, generation, "")
	if err != nil {
		allErrors = append(allErrors, err)
	}
	return utilerrors.NewAggregate(allErrors)
}

func (r *ControlPlaneReconciler) pruneResources(gvks []schema.GroupVersionKind, instanceGeneration string, namespace string) error {
	allErrors := []error{}
	labelSelector := map[string]string{common.OwnerKey: r.Instance.Namespace}
	for _, gvk := range gvks {
		if !strings.HasSuffix(gvk.Kind, "List") {
			gvk.Kind = gvk.Kind + "List"
		}
		list, err := r.Scheme.New(gvk)
		if err != nil {
			ulist := &unstructured.UnstructuredList{}
			ulist.SetGroupVersionKind(gvk)
			list = ulist
		}
		err = r.Client.List(context.TODO(), client.MatchingLabels(labelSelector).InNamespace(namespace), list)
		if err != nil {
			if !meta.IsNoMatchError(err) && !errors.IsNotFound(err) {
				r.Log.Error(err, "Error retrieving resources to prune", "type", gvk.String())
				allErrors = append(allErrors, err)
			}
			continue
		}
		objects, err := meta.ExtractList(list)
		if err != nil {
			r.Log.Error(err, "Error extracting list items for kind: %s", gvk.String())
			allErrors = append(allErrors, err)
			continue
		}
		for _, object := range objects {
			var resourceKey v1.ResourceKey
			metaObject, err := meta.Accessor(object)
			if err == nil {
				if typeObject, err := meta.TypeAccessor(object); err == nil {
					resourceKey = v1.NewResourceKey(metaObject, typeObject)
				} else {
					// we really should never get here
					r.Log.Error(err, "resource in %s does not implement Kubernetes ObjectMetadata", gvk.String())
					resourceKey = "unknown/unknown=unknown,Kind=unknown"
					}
			} else {
					// we really should never get here
				r.Log.Error(err, "resource in %s does not implement Kubernetes ObjectMetadata", gvk.String())
				continue
			}
			if generation, ok := common.GetAnnotation(metaObject, common.MeshGenerationKey); ok && generation != instanceGeneration {
				r.Log.Info("pruning resource", "resource", resourceKey)
				err = r.Client.Delete(context.TODO(), object, client.PropagationPolicy(metav1.DeletePropagationBackground))
				if err != nil && !errors.IsNotFound(err) {
					r.Log.Error(err, "Error pruning resource", "resource", resourceKey)
					allErrors = append(allErrors, err)
				} else {
					r.processDeletedObject(object)
				}
			}
		}
	}
	return utilerrors.NewAggregate(allErrors)
}
