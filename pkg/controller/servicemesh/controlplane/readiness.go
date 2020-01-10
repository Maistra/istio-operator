package controlplane

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/maistra/istio-operator/pkg/controller/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (r *ControlPlaneReconciler) UpdateReadiness() error {
	update, err := r.updateReadinessStatus()
	if update && !r.skipStatusUpdate() {
		statusErr := r.PostStatus()
		if statusErr != nil {
			// original error is more important than the status update error
			if err == nil {
				// if there's no original error, we can return the status update error
				return statusErr
			}
			// otherwise, we must log the status update error and return the original error
			log.Error(statusErr, "Error updating status")
		}
	}
	return err
}

func (r *ControlPlaneReconciler) updateReadinessStatus() (bool, error) {
	r.Log.Info("Updating ServiceMeshControlPlane readiness state")
	notReadyState, err := r.calculateNotReadyState()
	if err != nil {
		condition := v1.Condition{
			Type:    v1.ConditionTypeReady,
			Status:  v1.ConditionStatusUnknown,
			Reason:  v1.ConditionReasonProbeError,
			Message: fmt.Sprintf("Error collecting ready state: %s", err),
		}
		r.Status.SetCondition(condition)
		r.Manager.GetRecorder(controllerName).Event(r.Instance, corev1.EventTypeWarning, eventReasonNotReady, condition.Message)
		return true, err
	}
	unreadyComponents := make([]string, 0, len(notReadyState))
	for component, notReady := range notReadyState {
		if notReady {
			r.Log.Info(fmt.Sprintf("%s resources are not fully available", component))
			unreadyComponents = append(unreadyComponents, component)
		}
	}
	readyCondition := r.Status.GetCondition(v1.ConditionTypeReady)
	updateStatus := false
	if len(unreadyComponents) > 0 {
		if readyCondition.Status != v1.ConditionStatusFalse {
			condition := v1.Condition{
				Type:    v1.ConditionTypeReady,
				Status:  v1.ConditionStatusFalse,
				Reason:  v1.ConditionReasonComponentsNotReady,
				Message: "Some components are not fully available",
			}
			r.Status.SetCondition(condition)
			r.Manager.GetRecorder(controllerName).Event(r.Instance, corev1.EventTypeWarning, eventReasonNotReady, fmt.Sprintf("The following components are not fully available: %s", unreadyComponents))
			updateStatus = true
		}
	} else {
		if readyCondition.Status != v1.ConditionStatusTrue {
			condition := v1.Condition{
				Type:    v1.ConditionTypeReady,
				Status:  v1.ConditionStatusTrue,
				Reason:  v1.ConditionReasonComponentsReady,
				Message: "All component deployments are Available",
			}
			r.Status.SetCondition(condition)
			r.Manager.GetRecorder(controllerName).Event(r.Instance, corev1.EventTypeNormal, eventReasonReady, condition.Message)
			updateStatus = true
		}
	}

	return updateStatus, nil
}

func (r *ControlPlaneReconciler) calculateNotReadyState() (map[string]bool, error) {
	var cniNotReady bool
	notReadyState := map[string]bool{}
	err := r.calculateNotReadyStateForType(appsv1.SchemeGroupVersion.WithKind("Deployment"), notReadyState)
	if err != nil {
		return notReadyState, err
	}
	err = r.calculateNotReadyStateForType(appsv1.SchemeGroupVersion.WithKind("StatefulSet"), notReadyState)
	if err != nil {
		return notReadyState, err
	}
	err = r.calculateNotReadyStateForType(appsv1.SchemeGroupVersion.WithKind("DaemonSet"), notReadyState)
	if err != nil {
		return notReadyState, err
	}
	cniNotReady, err = r.calculateNotReadyStateForCNI()
	notReadyState["cni"] = cniNotReady
	return notReadyState, err
}

func (r *ControlPlaneReconciler) calculateNotReadyStateForCNI() (bool, error) {
	if !common.IsCNIEnabled {
		return false, nil
	}
	labelSelector := map[string]string{"istio": "cni"}
	daemonSets := &appsv1.DaemonSetList{}
	operatorNamespace := common.GetOperatorNamespace()
	if err := r.Client.List(context.TODO(), client.MatchingLabels(labelSelector).InNamespace(operatorNamespace), daemonSets); err != nil {
		return true, err
	}
	for _, ds := range daemonSets.Items {
		if !r.daemonSetReady(&ds) {
			return true, nil
		}
	}
	return false, nil
}

func (r *ControlPlaneReconciler) calculateNotReadyStateForType(gvk schema.GroupVersionKind, notReadyState map[string]bool) error {
	resources, err := r.FetchOwnedResources(gvk, r.Instance.GetNamespace(), r.Instance.GetNamespace())
	if err != nil {
		return err
	}
	for _, resource := range resources {
		ready := false
		var metaResource *metav1.ObjectMeta
		switch typedObject := resource.(type) {
		case *appsv1.DaemonSet:
			ready = r.daemonSetReady(typedObject)
			metaResource = &typedObject.ObjectMeta
		case *appsv1.StatefulSet:
			ready = r.statefulSetReady(typedObject)
			metaResource = &typedObject.ObjectMeta
		case *appsv1.Deployment:
			ready = r.deploymentReady(typedObject)
			metaResource = &typedObject.ObjectMeta
		default:
			r.Log.Error(nil, "skipping resource for readiness check: unknown resource type: %s", gvk.Kind)
			continue
		}
		if component, ok := common.GetLabel(metaResource, common.KubernetesAppComponentKey); ok {
			notReadyState[component] = notReadyState[component] || !ready
		} else {
			// how do we have an owned resource with no component label?
			r.Log.Error(nil, "skipping resource for readiness check: resource has no component label", gvk.Kind, metaResource.GetName())
		}
	}
	return nil
}

func (r *ControlPlaneReconciler) deploymentReady(deployment *appsv1.Deployment) bool {
	conditions := deployment.Status.Conditions
	for _, condition := range conditions {
		if condition.Type == appsv1.DeploymentAvailable {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}

func (r *ControlPlaneReconciler) statefulSetReady(statefulSet *appsv1.StatefulSet) bool {
	return statefulSet.Status.ReadyReplicas >= statefulSet.Status.Replicas
}

func (r *ControlPlaneReconciler) daemonSetReady(daemonSet *appsv1.DaemonSet) bool {
	return daemonSet.Status.NumberUnavailable == 0
}
