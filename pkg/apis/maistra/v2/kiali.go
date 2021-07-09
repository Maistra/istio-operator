package v2

import corev1 "k8s.io/api/core/v1"

// KialiAddonConfig is used to configure a kiali instance for use with the mesh
type KialiAddonConfig struct {
	Enablement `json:",inline"`
	// Name of Kiali CR, Namespace must match control plane namespace
	Name string `json:"name,omitempty"`
	// Install a Kiali resource if the named Kiali resource is not present.
	// +optional
	Install *KialiInstallConfig `json:"install,omitempty"`
}

// KialiInstallConfig is used to configure a kiali installation
type KialiInstallConfig struct {
	// Dashboard configures the behavior of the kiali dashboard.
	// +optional
	Dashboard *KialiDashboardConfig `json:"dashboard,omitempty"`
	// Service is used to configure the k8s Service associated with the kiali
	// installation.
	// XXX: provided for upstream support, only ingress is used, and then only
	// for enablement and contextPath
	// +optional
	Service *ComponentServiceConfig `json:"service,omitempty"`

	// Deployment configures the kiali deployment.
	// +optional
	// +deprecated
	Deployment *KialiDeploymentConfig `json:"deployment,omitempty"`
}

// KialiDashboardConfig configures the behavior of the kiali dashboard
type KialiDashboardConfig struct {
	// ViewOnly configures view_only_mode for the dashboard
	// .Values.kiali.dashboard.viewOnlyMode
	// +optional
	ViewOnly *bool `json:"viewOnly,omitempty"`
	// XXX: should the user have a choice here, or should these be configured
	// automatically if they are enabled for the control plane installation?
	// Grafana endpoint will be configured based on Grafana configuration
	// +optional
	EnableGrafana *bool `json:"enableGrafana,omitempty"`
	// Prometheus endpoint will be configured based on Prometheus configuration
	// +optional
	EnablePrometheus *bool `json:"enablePrometheus,omitempty"`
	// Tracing endpoint will be configured based on Tracing configuration
	// +optional
	EnableTracing *bool `json:"enableTracing,omitempty"`
}

// DEPRECATED: Use runtime.components.kiali instead.
// KialiDeploymentConfig configures the kiali deployment
// +deprecated
type KialiDeploymentConfig struct {
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the kiali pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// ResourceName returns the resource name for the Kiali resource, returning a
// sensible default if the Name field is not set ("kiali").
func (c KialiAddonConfig) ResourceName() string {
	if c.Name == "" {
		return "kiali"
	}
	return c.Name
}
