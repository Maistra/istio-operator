package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// StatusType represents the status for a control plane, component, or resource
type StatusType struct {
	Resource           string      `json:"resource,omitempty"`
	ObservedGeneration int64       `json:"observedGeneration,omitempty"`
	ReconciledVersion  string      `json:"reconciledVersion,omitempty"`
	Conditions         []Condition `json:"conditions,omitempty"`
}

// NewStatus returns a new StatusType object
func NewStatus() StatusType {
	return StatusType{Conditions: make([]Condition, 0, 3)}
}

// NewControlPlaneStatus returns an initialized ControlPlaneStatus object
func NewControlPlaneStatus() *ControlPlaneStatus {
	return &ControlPlaneStatus{ComponentStatus: []*ComponentStatus{}}
}

// FindComponentByName returns the status for a specific component
func (s *ControlPlaneStatus) FindComponentByName(name string) *ComponentStatus {
	for _, status := range s.ComponentStatus {
		if status.Resource == name {
			return status
		}
	}
	return nil
}

// NewComponentStatus returns a new ComponentStatus object
func NewComponentStatus() *ComponentStatus {
	return &ComponentStatus{StatusType: NewStatus(), Resources: []*StatusType{}}
}

// ComponentStatus represents the status of an object with children
type ComponentStatus struct {
	StatusType `json:",inline"`
	Resources  []*StatusType `json:"children,omitempty"`
}

// ConditionType represents the type of the condition.  Condition stages are:
// Installed, Reconciled, Ready
type ConditionType string

const (
	// ConditionTypeInstalled signifies the whether or not the controller has
	// installed the resources defined through the CR.
	ConditionTypeInstalled ConditionType = "Installed"
	// ConditionTypeReconciled signifies the whether or not the controller has
	// reconciled the resources defined through the CR.
	ConditionTypeReconciled ConditionType = "Reconciled"
	// ConditionTypeReady signifies the whether or not any Deployment, StatefulSet,
	// etc. resources are Ready.
	ConditionTypeReady ConditionType = "Ready"
)

// ConditionStatus represents the status of the condition
type ConditionStatus string

const (
	// ConditionStatusTrue represents completion of the condition, e.g.
	// Initialized=True signifies that initialization has occurred.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents incomplete status of the condition, e.g.
	// Initialized=False signifies that initialization has not occurred or has
	// failed.
	ConditionStatusFalse ConditionStatus = "False"
	// ConditionStatusUnknown represents unknown completion of the condition, e.g.
	// Initialized=Unknown signifies that initialization may or may not have been
	// completed.
	ConditionStatusUnknown ConditionStatus = "Unknown"
)

// ConditionReason represents a short message indicating how the condition came
// to be in its present state.
type ConditionReason string

const (
	// ConditionReasonDeletionError ...
	ConditionReasonDeletionError ConditionReason = "DeletionError"
	// ConditionReasonInstallSuccessful ...
	ConditionReasonInstallSuccessful ConditionReason = "InstallSuccessful"
	// ConditionReasonInstallError ...
	ConditionReasonInstallError ConditionReason = "InstallError"
	// ConditionReasonReconcileSuccessful ...
	ConditionReasonReconcileSuccessful ConditionReason = "ReconcileSuccessful"
	// ConditionReasonReconcileError ...
	ConditionReasonReconcileError ConditionReason = "ReconcileError"
	// ConditionReasonResourceCreated ...
	ConditionReasonResourceCreated ConditionReason = "ResourceCreated"
	// ConditionReasonSpecUpdated ...
	ConditionReasonSpecUpdated ConditionReason = "SpecUpdated"
	// ConditionReasonUpdateSuccessful ...
	ConditionReasonUpdateSuccessful ConditionReason = "UpdateSuccessful"
	// ConditionReasonComponentsReady ...
	ConditionReasonComponentsReady ConditionReason = "ComponentsReady"
	// ConditionReasonComponentsNotReady ...
	ConditionReasonComponentsNotReady ConditionReason = "ComponentsNotReady"
	// ConditionReasonProbeError ...
	ConditionReasonProbeError ConditionReason = "ProbeError"
	// ConditionReasonPausingInstall ...
	ConditionReasonPausingInstall ConditionReason = "PausingInstall"
	// ConditionReasonPausingUpdate ...
	ConditionReasonPausingUpdate ConditionReason = "PausingUpdate"
	// ConditionReasonDeleting ...
	ConditionReasonDeleting ConditionReason = "Deleting"
	// ConditionReasonDeleted ...
	ConditionReasonDeleted ConditionReason = "Deleted"
)

// Condition represents a specific condition on a resource
type Condition struct {
	Type               ConditionType   `json:"type,omitempty"`
	Status             ConditionStatus `json:"status,omitempty"`
	Reason             ConditionReason `json:"reason,omitempty"`
	Message            string          `json:"message,omitempty"`
	LastTransitionTime metav1.Time     `json:"lastTransitionTime,omitempty"`
}

// GetReconciledVersion returns the reconciled version, or a default for older resources
func (s *StatusType) GetReconciledVersion() string {
	if s == nil {
		return "0.0.0-0"
	}
	if s.ReconciledVersion == "" {
		return fmt.Sprintf("1.0.0-%d", s.ObservedGeneration)
	}
	return s.ReconciledVersion
}

// GetCondition removes a condition for the list of conditions
func (s *StatusType) GetCondition(conditionType ConditionType) Condition {
	if s == nil {
		return Condition{Type: conditionType, Status: ConditionStatusUnknown}
	}
	for i := range s.Conditions {
		if s.Conditions[i].Type == conditionType {
			return s.Conditions[i]
		}
	}
	return Condition{Type: conditionType, Status: ConditionStatusUnknown}
}

// SetCondition sets a specific condition in the list of conditions
func (s *StatusType) SetCondition(condition Condition) *StatusType {
	if s == nil {
		return nil
	}
	now := metav1.Now()
	for i := range s.Conditions {
		if s.Conditions[i].Type == condition.Type {
			if s.Conditions[i].Status != condition.Status {
				condition.LastTransitionTime = now
			} else {
				condition.LastTransitionTime = s.Conditions[i].LastTransitionTime
			}
			s.Conditions[i] = condition
			return s
		}
	}

	// If the condition does not exist,
	// initialize the lastTransitionTime
	condition.LastTransitionTime = now
	s.Conditions = append(s.Conditions, condition)
	return s
}

// RemoveCondition removes a condition for the list of conditions
func (s *StatusType) RemoveCondition(conditionType ConditionType) *StatusType {
	if s == nil {
		return nil
	}
	for i := range s.Conditions {
		if s.Conditions[i].Type == conditionType {
			s.Conditions = append(s.Conditions[:i], s.Conditions[i+1:]...)
			return s
		}
	}
	return s
}

// ResourceKey is a typedef for key used in ManagedGenerations.  It is a string
// with the format: namespace/name=group/version,kind
type ResourceKey string

// NewResourceKey for the object and type
func NewResourceKey(o metav1.Object, t metav1.Type) ResourceKey {
	return ResourceKey(fmt.Sprintf("%s/%s=%s,Kind=%s", o.GetNamespace(), o.GetName(), t.GetAPIVersion(), t.GetKind()))
}

// ToUnstructured returns a an Unstructured object initialized with Namespace,
// Name, APIVersion, and Kind fields from the ResourceKey
func (key ResourceKey) ToRuntimeObject(scheme *runtime.Scheme) (runtime.Object, error) {
	// ResourceKey is guaranteed to be at least "/=," meaning we are guaranteed
	// to get two elements in all of the splits
	parts := strings.SplitN(string(key), "=", 2)
	nn := strings.SplitN(parts[0], "/", 2)
	gvk := strings.SplitN(parts[1], ",Kind=", 2)
	retval, err := scheme.New(schema.FromAPIVersionAndKind(gvk[0], gvk[1]))
	if err != nil {
		unst := &unstructured.Unstructured{}
		unst.SetNamespace(nn[0])
		unst.SetName(nn[1])
		unst.SetAPIVersion(gvk[0])
		unst.SetKind(gvk[1])
		return unst, nil
	}
	if metaObject, err := meta.Accessor(retval); err == nil {
		metaObject.SetNamespace(nn[0])
		metaObject.SetName(nn[1])
	} else {
		return nil, err
	}
	return retval, nil
}

// FindResourcesOfKind returns all the specified kind.  Note, this does not account for group or version.
func (s *ComponentStatus) FindResourcesOfKind(kind string) []*StatusType {
	resources := []*StatusType{}
	suffix := ",Kind=" + kind
	for _, status := range s.Resources {
		if strings.HasSuffix(status.Resource, suffix) {
			resources = append(resources, status)
		}
	}
	return resources
}

// FindResourceByKey returns the status for a specific child resource
func (s *ComponentStatus) FindResourceByKey(key ResourceKey) *StatusType {
	for _, status := range s.Resources {
		if status.Resource == string(key) {
			return status
		}
	}
	return nil
}
