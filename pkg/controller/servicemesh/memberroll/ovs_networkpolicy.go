package memberroll

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/maistra/istio-operator/pkg/controller/common"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type networkPolicyStrategy struct {
	common.ResourceManager
	meshNamespace           string
	requiredNetworkPolicies sets.String
	networkPoliciesList     []*networkingv1.NetworkPolicy
}

var _ networkingStrategy = (*networkPolicyStrategy)(nil)

func newNetworkPolicyStrategy(r *namespaceReconciler) (*networkPolicyStrategy, error) {
	var err error
	strategy := &networkPolicyStrategy{
		ResourceManager:         r.ResourceManager,
		meshNamespace:           r.meshNamespace,
		requiredNetworkPolicies: sets.NewString(),
	}
	strategy.Log = r.Log.WithValues("NetworkStrategy", "NetworkPolicy")
	networkPolicies, err := r.FetchOwnedResources(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"), strategy.meshNamespace, strategy.meshNamespace)
	if err != nil {
		strategy.Log.Error(err, "error retrieving NetworkPolicy resources for mesh")
		return nil, err
	}
	for _, obj := range networkPolicies {
		if np, ok := obj.(*networkingv1.NetworkPolicy); ok {
			strategy.networkPoliciesList = append(strategy.networkPoliciesList, np)
			if metav1.HasAnnotation(np.ObjectMeta, common.InternalKey) {
				continue
			}
			strategy.requiredNetworkPolicies.Insert(np.GetName())
		} else {
			err = fmt.Errorf("runtime.Object from NetworkPolicyList is not a NetworkPolicy: %s", obj.GetObjectKind().GroupVersionKind().String())
			strategy.Log.Error(err, "runtim.Object is not a NetworkPolicy")
			return nil, err
		}
	}
	return strategy, nil
}

func (s *networkPolicyStrategy) reconcileNamespaceInMesh(namespace string) error {
	logger := s.Log.WithValues("Namespace", namespace)
	namespaceNetworkPolicies, err := s.FetchMeshResources(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"), s.meshNamespace, namespace)
	if err != nil {
		logger.Error(err, "error retrieving NetworkPolicy resources for namespace")
		return err
	}

	allErrors := []error{}

	// add required network policies
	existingNetworkPolicies := nameSetFromSlice(namespaceNetworkPolicies)
	addedNetworkPolicies := sets.NewString()
	for _, meshNetworkPolicy := range s.networkPoliciesList {
		networkPolicyName := meshNetworkPolicy.GetName()
		if !s.requiredNetworkPolicies.Has(networkPolicyName) {
			// this is not required for members
			continue
		}
		if !existingNetworkPolicies.Has(networkPolicyName) {
			logger.Info("creating NetworkPolicy", "NetworkPolicy", networkPolicyName)
			networkPolicy := meshNetworkPolicy.DeepCopy()
			networkPolicy.SetNamespace(namespace)
			common.SetLabel(networkPolicy, common.MemberOfKey, s.meshNamespace)
			err = s.Client.Create(context.TODO(), networkPolicy)
			if err == nil {
				addedNetworkPolicies.Insert(networkPolicyName)
			} else {
				logger.Error(err, "error creating NetworkPolicy", "NetworkPolicy", networkPolicyName)
				allErrors = append(allErrors, err)
			}
		} // XXX: else if existingNetworkPolicy.annotations[mesh-generation] != meshNetworkPolicy.annotations[generation] then update?
	}

	existingNetworkPolicies = existingNetworkPolicies.Union(addedNetworkPolicies)

	// delete obsolete network policies
	for networkPolicyName := range existingNetworkPolicies.Difference(s.requiredNetworkPolicies) {
		logger.Info("deleting NetworkPolicy", "NetworkPolicy", networkPolicyName)
		networkPolicy := &networkingv1.NetworkPolicy{}
		networkPolicy.SetGroupVersionKind(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"))
		networkPolicy.SetName(networkPolicyName)
		networkPolicy.SetNamespace(namespace)
		err = s.Client.Delete(context.TODO(), networkPolicy, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil && !(errors.IsNotFound(err) || errors.IsGone(err)) {
			logger.Error(err, "error deleting NetworkPolicy", "NetworkPolicy", networkPolicyName)
			allErrors = append(allErrors, err)
		}
	}

	// if there were errors, we've logged them and there's not really anything we can do, as we're in an uncertain state
	// maybe a following reconcile will add the required network policy that failed.  if it was a delete that failed, we're
	// just leaving behind some cruft.
	return utilerrors.NewAggregate(allErrors)
}

func (s *networkPolicyStrategy) removeNamespaceFromMesh(namespace string) error {
	allErrors := []error{}
	logger := s.Log.WithValues("Namespace", namespace)
	npList, err := s.FetchMeshResources(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"), s.meshNamespace, namespace)
	if err == nil {
		for _, obj := range npList {
			if np, ok := obj.(*networkingv1.NetworkPolicy); ok {
				logger.Info("deleting NetworkPolicy for mesh", "NetworkPolicy", np.GetName())
				err = s.Client.Delete(context.TODO(), np)
				if err != nil {
					logger.Error(err, "error removing NetworkPolicy associated with mesh", "NetworkPolicy", np.GetName())
					allErrors = append(allErrors, err)
				}
			} else {
				err = fmt.Errorf("runtime.Object from NetworkPolicyList is not a NetworkPolicy: %s", obj.GetObjectKind().GroupVersionKind().String())
				s.Log.Error(err, "runtim.Object is not a NetworkPolicy")
			}
		}
	} else {
		logger.Error(err, "error could not retrieve NetworkPolicy resources associated with mesh")
		allErrors = append(allErrors, err)
	}
	return utilerrors.NewAggregate(allErrors)
}
