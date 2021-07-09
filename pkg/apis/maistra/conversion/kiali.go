package conversion

import (
	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	corev1 "k8s.io/api/core/v1"
)

func populateKialiAddonValues(kiali *v2.KialiAddonConfig, values map[string]interface{}) (reterr error) {
	if kiali == nil {
		return nil
	}

	kialiValues := make(map[string]interface{})
	if kiali.Name != "" {
		if err := setHelmStringValue(kialiValues, "resourceName", kiali.Name); err != nil {
			return err
		}
	}
	if kiali.Enabled != nil {
		if err := setHelmBoolValue(kialiValues, "enabled", *kiali.Enabled); err != nil {
			return err
		}
	}
	defer func() {
		if reterr == nil {
			if len(kialiValues) > 0 {
				if err := overwriteHelmValues(values, kialiValues, "kiali"); err != nil {
					reterr = err
				}
			}
		}
	}()

	if kiali.Install == nil {
		return nil
	}

	dashboardConfig := kiali.Install.Dashboard
	if dashboardConfig != nil {
		if dashboardConfig.ViewOnly != nil {
			if err := setHelmBoolValue(kialiValues, "dashboard.viewOnlyMode", *dashboardConfig.ViewOnly); err != nil {
				return err
			}
		}
		if dashboardConfig.EnableGrafana != nil {
			if err := setHelmBoolValue(kialiValues, "dashboard.enableGrafana", *dashboardConfig.EnableGrafana); err != nil {
				return err
			}
		}
		if dashboardConfig.EnablePrometheus != nil {
			if err := setHelmBoolValue(kialiValues, "dashboard.enablePrometheus", *dashboardConfig.EnablePrometheus); err != nil {
				return err
			}
		}
		if dashboardConfig.EnableTracing != nil {
			if err := setHelmBoolValue(kialiValues, "dashboard.enableTracing", *dashboardConfig.EnableTracing); err != nil {
				return err
			}
		}
	}

	deploymentConfig := kiali.Install.Deployment
	if deploymentConfig != nil {
		resources := deploymentConfig.Resources
		if resources != nil {
			if values, err := toValues(resources); err == nil {
				if len(values) > 0 {
					if err := setHelmValue(kialiValues, "deployment_resources", values); err != nil {
						return err
					}
				}
			}
		}

		affinity := deploymentConfig.Affinity
		if affinity != nil {
			nodeAffinity := affinity.NodeAffinity
			if nodeAffinity != nil {
				if values, err := toValues(nodeAffinity); err == nil {
					if len(values) > 0 {
						if err := setHelmValue(kialiValues, "deployment_affinity.node", values); err != nil {
							return err
						}
					}
				}
			}

			podAffinity := affinity.PodAffinity
			if podAffinity != nil {
				if values, err := toValues(podAffinity); err == nil {
					if len(values) > 0 {
						if err := setHelmValue(kialiValues, "deployment_affinity.pod", values); err != nil {
							return err
						}
					}
				}
			}

			podAntiAffinity := affinity.PodAntiAffinity
			if podAntiAffinity != nil {
				if values, err := toValues(podAntiAffinity); err == nil {
					if len(values) > 0 {
						if err := setHelmValue(kialiValues, "deployment_affinity.pod_anti", values); err != nil {
							return err
						}
					}
				}
			}
		}

		nodeSelector := deploymentConfig.NodeSelector
		if nodeSelector != nil {
			if values, err := toValues(nodeSelector); err == nil {
				if len(values) > 0 {
					if err := setHelmValue(kialiValues, "deployment_nodeSelector", values); err != nil {
						return err
					}
				}
			}
		}

		if len(deploymentConfig.Tolerations) > 0 {
			untypedSlice := make([]interface{}, len(deploymentConfig.Tolerations))
			for index, toleration := range deploymentConfig.Tolerations {
				untypedSlice[index] = toleration
			}
			if tolerations, err := sliceToValues(untypedSlice); err != nil {
				return err
			} else if len(tolerations) > 0 {
				if err := setHelmValue(kialiValues, "deployment_tolerations", tolerations); err != nil {
					return err
				}
			}
		}
	}

	if kiali.Install.Service != nil {
		if err := populateComponentServiceValues(kiali.Install.Service, kialiValues); err != nil {
			return err
		}
	}
	if rawContextPath, ok := kialiValues["contextPath"]; ok {
		if contextPath, ok := rawContextPath.(string); ok {
			if err := setHelmStringValue(kialiValues, "contextPath", contextPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func populateKialiAddonConfig(in *v1.HelmValues, out *v2.KialiAddonConfig) (bool, error) {
	rawKialiValues, ok, err := in.GetMap("kiali")
	if err != nil {
		return false, err
	} else if !ok || len(rawKialiValues) == 0 {
		return false, nil
	}

	// remove values not configured through kiali addon
	kialiValues := v1.NewHelmValues(rawKialiValues).DeepCopy()
	delete(kialiValues.GetContent(), "prometheusAddr")
	if dashboardValues, ok, err := kialiValues.GetMap("dashboard"); ok {
		delete(dashboardValues, "grafanaURL")
		if len(dashboardValues) == 0 {
			delete(kialiValues.GetContent(), "dashboard")
		}
	} else if err != nil {
		return false, nil
	}
	if len(kialiValues.GetContent()) == 0 {
		return false, nil
	}
	// we want to use the original, now that we're checking to see if there is actual kiali config
	kialiValues = v1.NewHelmValues(rawKialiValues)
	setKiali := false

	kiali := out

	if name, ok, err := kialiValues.GetAndRemoveString("resourceName"); ok {
		kiali.Name = name
		setKiali = true
	} else if err != nil {
		return false, err
	}

	if enabled, ok, err := kialiValues.GetAndRemoveBool("enabled"); ok {
		kiali.Enabled = &enabled
		setKiali = true
	} else if err != nil {
		return false, err
	}

	install := &v2.KialiInstallConfig{}
	setInstall := false

	dashboardConfig := &v2.KialiDashboardConfig{}
	setDashboard := false
	if viewOnlyMode, ok, err := kialiValues.GetAndRemoveBool("dashboard.viewOnlyMode"); ok {
		dashboardConfig.ViewOnly = &viewOnlyMode
		setDashboard = true
	} else if err != nil {
		return false, err
	}
	if enableGrafana, ok, err := kialiValues.GetAndRemoveBool("dashboard.enableGrafana"); ok {
		dashboardConfig.EnableGrafana = &enableGrafana
		setDashboard = true
	} else if err != nil {
		return false, err
	}
	if enablePrometheus, ok, err := kialiValues.GetAndRemoveBool("dashboard.enablePrometheus"); ok {
		dashboardConfig.EnablePrometheus = &enablePrometheus
		setDashboard = true
	} else if err != nil {
		return false, err
	}
	if enableTracing, ok, err := kialiValues.GetAndRemoveBool("dashboard.enableTracing"); ok {
		dashboardConfig.EnableTracing = &enableTracing
		setDashboard = true
	} else if err != nil {
		return false, err
	}
	if setDashboard == true {
		setInstall = true
		install.Dashboard = dashboardConfig
	}

	service := &v2.ComponentServiceConfig{}
	if applied, err := populateComponentServiceConfig(kialiValues, service); applied {
		setInstall = true
		install.Service = service
	} else if err != nil {
		return false, err
	}
	if install.Service == nil || install.Service.Ingress == nil || install.Service.Ingress.ContextPath == "" {
		// check old kiali.contextPath
		if contextPath, ok, err := kialiValues.GetAndRemoveString("contextPath"); ok && contextPath != "" {
			if install.Service.Ingress == nil {
				install.Service.Ingress = &v2.ComponentIngressConfig{}
			}
			install.Service.Ingress.ContextPath = contextPath
			setInstall = true
		} else if err != nil {
			return false, err
		}
	}

	deploymentConfig := &v2.KialiDeploymentConfig{}
	setDeployment := false

	if resources, ok, err := kialiValues.GetAndRemoveMap("deployment_resources"); ok {
		if len(resources) > 0 {
			deploymentConfig.Resources = &corev1.ResourceRequirements{}
			if err := fromValues(resources, deploymentConfig.Resources); err != nil {
				return false, err
			}
			setDeployment = true
		}
	} else if err != nil {
		return false, err
	}

	affinity := &corev1.Affinity{}
	setDeploymentAffinity := false

	if nodeAffinity, ok, err := kialiValues.GetAndRemoveMap("deployment_affinity.node"); ok {
		if len(nodeAffinity) > 0 {
			affinity.NodeAffinity = &corev1.NodeAffinity{}
			if err := fromValues(nodeAffinity, affinity.NodeAffinity); err != nil {
				return false, err
			}
			setDeploymentAffinity = true
		}
	} else if err != nil {
		return false, err
	}

	if podAffinity, ok, err := kialiValues.GetAndRemoveMap("deployment_affinity.pod"); ok {
		if len(podAffinity) > 0 {
			affinity.PodAffinity = &corev1.PodAffinity{}
			if err := fromValues(podAffinity, affinity.PodAffinity); err != nil {
				return false, err
			}
			setDeploymentAffinity = true
		}
	} else if err != nil {
		return false, err
	}

	if podAntiAffinity, ok, err := kialiValues.GetAndRemoveMap("deployment_affinity.pod_anti"); ok {
		if len(podAntiAffinity) > 0 {
			affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			if err := fromValues(podAntiAffinity, affinity.PodAntiAffinity); err != nil {
				return false, err
			}
			setDeploymentAffinity = true
		}
	} else if err != nil {
		return false, err
	}

	if setDeploymentAffinity {
		setDeployment = true
		deploymentConfig.Affinity = affinity
	}

	if nodeSelector, ok, err := kialiValues.GetAndRemoveStringMap("deployment_nodeSelector"); ok {
		if len(nodeSelector) > 0 {
			deploymentConfig.NodeSelector = nodeSelector
			setDeployment = true
		}
	} else if err != nil {
		return false, err
	}

	if tolerations, ok, err := kialiValues.GetAndRemoveSlice("deployment_tolerations"); ok && len(tolerations) > 0 {
		deploymentConfig.Tolerations = make([]corev1.Toleration, len(tolerations))
		setDeployment = true
		for index, tolerationValues := range tolerations {
			if err := fromValues(tolerationValues, &deploymentConfig.Tolerations[index]); err != nil {
				return false, err
			}
		}
	} else if err != nil {
		return false, err
	}

	if setDeployment {
		setInstall = true
		install.Deployment = deploymentConfig
	}

	if setInstall {
		kiali.Install = install
		setKiali = true
	}
	// update the kiali settings
	if len(kialiValues.GetContent()) == 0 {
		in.RemoveField("kiali")
	} else if err := in.SetField("kiali", kialiValues.GetContent()); err != nil {
		return false, err
	}

	return setKiali, nil
}
