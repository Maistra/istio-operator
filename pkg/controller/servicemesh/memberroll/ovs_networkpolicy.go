package memberroll

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/maistra/istio-operator/pkg/controller/common"

	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type networkPolicyStrategy struct {
	common.ControllerResources
	meshNamespace           string
	requiredNetworkPolicies sets.String
	networkPoliciesList     *networking.NetworkPolicyList
}

var _ NamespaceReconciler = (*networkPolicyStrategy)(nil)

func newNetworkPolicyStrategy(cl client.Client, baseLogger logr.Logger, meshNamespace string) (*networkPolicyStrategy, error) {
	strategy := &networkPolicyStrategy{
		ControllerResources: common.ControllerResources{
			Client: cl,
			Log:    baseLogger.WithValues("NetworkStrategy", "NetworkPolicy"),
		},
		meshNamespace:           meshNamespace,
		requiredNetworkPolicies: sets.NewString(),
	}
	strategy.networkPoliciesList = &networking.NetworkPolicyList{}
	labelSelector := map[string]string{common.OwnerKey: strategy.meshNamespace}
	err := cl.List(context.TODO(), client.MatchingLabels(labelSelector).InNamespace(strategy.meshNamespace), strategy.networkPoliciesList)
	if err != nil {
		strategy.Log.Error(err, "error retrieving NetworkPolicy resources for mesh")
		return nil, err
	}
	for _, np := range strategy.networkPoliciesList.Items {
		if _, ok := common.GetAnnotation(&np, common.InternalKey); ok {
			continue
		}
		strategy.requiredNetworkPolicies.Insert(np.GetName())
	}
	return strategy, nil
}

func (s *networkPolicyStrategy) reconcileNamespaceInMesh(namespace string) error {
	logger := s.Log.WithValues("Namespace", namespace)

	namespaceNetworkPolicies := &networking.NetworkPolicyList{}
	labelSelector := map[string]string{common.MemberOfKey: s.meshNamespace}
	err := s.Client.List(context.TODO(), client.MatchingLabels(labelSelector).InNamespace(namespace), namespaceNetworkPolicies)
	if err != nil {
		logger.Error(err, "error retrieving NetworkPolicy resources for namespace")
		return err
	}

	allErrors := []error{}

	// add required network policies
	existingNetworkPolicies := nameSet(namespaceNetworkPolicies)
	addedNetworkPolicies := sets.NewString()
	for _, meshNetworkPolicy := range s.networkPoliciesList.Items {
		networkPolicyName := meshNetworkPolicy.GetName()
		if !s.requiredNetworkPolicies.Has(networkPolicyName) {
			// this is not required for members
			continue
		}
		if !existingNetworkPolicies.Has(networkPolicyName) {
			logger.Info("creating NetworkPolicy", "NetworkPolicy", networkPolicyName)
			networkPolicy := meshNetworkPolicy.DeepCopy()
			networkPolicy.ObjectMeta = meta.ObjectMeta{
				Name:        networkPolicyName,
				Namespace:   namespace,
				Labels:      copyMap(meshNetworkPolicy.Labels),
				Annotations: copyMap(meshNetworkPolicy.Annotations),
			}
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
		networkPolicy := &networking.NetworkPolicy{
			ObjectMeta: meta.ObjectMeta{
				Name:      networkPolicyName,
				Namespace: namespace,
			},
		}
		err = s.Client.Delete(context.TODO(), networkPolicy, client.PropagationPolicy(meta.DeletePropagationForeground))
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

	npList := &networking.NetworkPolicyList{}
	labelSelector := map[string]string{common.MemberOfKey: s.meshNamespace}
	err := s.Client.List(context.TODO(), client.MatchingLabels(labelSelector).InNamespace(namespace), npList)
	if err == nil {
		for _, np := range npList.Items {
			logger.Info("deleting NetworkPolicy for mesh", "NetworkPolicy", np.GetName())
			err = s.Client.Delete(context.TODO(), &np)
			if err != nil {
				logger.Error(err, "error removing NetworkPolicy associated with mesh", "NetworkPolicy", np.GetName())
				allErrors = append(allErrors, err)
			}
		}
	} else {
		logger.Error(err, "error could not retrieve NetworkPolicy resources associated with mesh")
		allErrors = append(allErrors, err)
	}
	return utilerrors.NewAggregate(allErrors)
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, val := range in {
		out[key] = val
	}
	return out
}
