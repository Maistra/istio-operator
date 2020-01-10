package memberroll

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	networkv1 "github.com/openshift/api/network/v1"
	"github.com/openshift/library-go/pkg/network/networkapihelpers"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type multitenantStrategy struct {
	client        client.Client
	logger        logr.Logger
	meshNamespace string
}

var _ networkingStrategy = (*multitenantStrategy)(nil)

func newMultitenantStrategy(r *namespaceReconciler) (*multitenantStrategy, error) {
	return &multitenantStrategy{
		client:                  r.Client,
		logger:                  r.Log.WithValues("NetworkStrategy", "Multitenant"),
		meshNamespace:           r.meshNamespace,
	}, nil
}

func (s *multitenantStrategy) reconcileNamespaceInMesh(namespace string) error {
	s.logger.Info("joining network to mesh", "Namespace", namespace)
	return s.updateNetworkNamespace(namespace, networkapihelpers.JoinPodNetwork, s.meshNamespace)
}

func (s *multitenantStrategy) removeNamespaceFromMesh(namespace string) error {
	s.logger.Info("isolating network", "Namespace", namespace)
	return s.updateNetworkNamespace(namespace, networkapihelpers.IsolatePodNetwork, "")
}

// adapted from github.com/openshift/oc/pkg/cli/admin/network/project_options.go#UpdatePodNetwork()
func (s *multitenantStrategy) updateNetworkNamespace(namespace string, action networkapihelpers.PodNetworkAction, args string) error {
	netns := &networkv1.NetNamespace{}
	netns.SetGroupVersionKind(networkv1.GroupVersion.WithKind("NetNamespace"))
	err := s.client.Get(context.TODO(), client.ObjectKey{Name: namespace}, netns)
	if err != nil {
		return err
	}
	networkapihelpers.SetChangePodNetworkAnnotation(netns, action, args)
	err = s.client.Update(context.TODO(), netns)
	if err != nil {
		return err
	}
	// Validate SDN controller applied or rejected the intent
	backoff := wait.Backoff{
		Steps:    15,
		Duration: 500 * time.Millisecond,
		Factor:   1.1,
	}
	return wait.ExponentialBackoff(backoff, func() (bool, error) {
		updatedNetNs := &networkv1.NetNamespace{}
		updatedNetNs.SetGroupVersionKind(networkv1.GroupVersion.WithKind("NetNamespace"))
		err := s.client.Get(context.TODO(), client.ObjectKey{Name: namespace}, updatedNetNs)
		if err != nil {
			return false, err
		}

		if _, _, err = networkapihelpers.GetChangePodNetworkAnnotation(updatedNetNs); err == networkapihelpers.ErrorPodNetworkAnnotationNotFound {
			return true, nil
		}
		// Pod network change not applied yet
		return false, nil
	})
}
