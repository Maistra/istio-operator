package conversion

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/versions"
)

var (
	grafanaTestAddress  = "grafana.other-namespace.svc.cluster.local:3001"
	grafanaTestNodePort = int32(12345)
)

var grafanaTestCases = []conversionTestCase{
	{
		name: "nil." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: nil,
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "enablement." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Enablement: v2.Enablement{
							Enabled: &featureEnabled,
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"enabled": true,
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "existing." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Address: &grafanaTestAddress,
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"dashboard": map[string]interface{}{
					"grafanaURL": "grafana.other-namespace.svc.cluster.local:3001",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.env." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Config: v2.GrafanaConfig{
								Env: map[string]string{
									"GF_SMTP_ENABLED": "true",
								},
								EnvSecrets: map[string]string{
									"GF_SMTP_USER": "grafana-secrets",
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"env": map[string]interface{}{
					"GF_SMTP_ENABLED": "true",
				},
				"envSecrets": map[string]interface{}{
					"GF_SMTP_USER": "grafana-secrets",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.persistence.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Persistence: &v2.ComponentPersistenceConfig{},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.persistence.simple." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Persistence: &v2.ComponentPersistenceConfig{
								Enablement: v2.Enablement{
									Enabled: &featureEnabled,
								},
								AccessMode:       corev1.ReadWriteOnce,
								StorageClassName: "standarad",
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"accessMode":       "ReadWriteOnce",
				"persist":          true,
				"storageClassName": "standarad",
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.persistence.resources.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Persistence: &v2.ComponentPersistenceConfig{
								Resources: &corev1.ResourceRequirements{},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.persistence.resources.values." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Persistence: &v2.ComponentPersistenceConfig{
								Resources: &corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("5Gi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("25Gi"),
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"persistenceResources": map[string]interface{}{
					"limits": map[string]interface{}{
						"storage": "25Gi",
					},
					"requests": map[string]interface{}{
						"storage": "5Gi",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.security.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Security: &v2.GrafanaSecurityConfig{},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.security.full." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Security: &v2.GrafanaSecurityConfig{
								Enablement: v2.Enablement{
									Enabled: &featureEnabled,
								},
								PassphraseKey: "passphrase",
								SecretName:    "htpasswd",
								UsernameKey:   "username",
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"security": map[string]interface{}{
					"enabled":       true,
					"passphraseKey": "passphrase",
					"secretName":    "htpasswd",
					"usernameKey":   "username",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.service.misc." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Service: v2.ComponentServiceConfig{
								Metadata: v2.MetadataConfig{
									Annotations: map[string]string{
										"some-service-annotation": "service-annotation-value",
									},
									Labels: map[string]string{
										"some-service-label": "service-label-value",
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"service": map[string]interface{}{
					"annotations": map[string]interface{}{
						"some-service-annotation": "service-annotation-value",
					},
					"labels": map[string]interface{}{
						"some-service-label": "service-label-value",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.service.ingress.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Service: v2.ComponentServiceConfig{
								Ingress: &v2.ComponentIngressConfig{},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.service.ingress.full." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Service: v2.ComponentServiceConfig{
								Ingress: &v2.ComponentIngressConfig{
									Enablement: v2.Enablement{
										Enabled: &featureEnabled,
									},
									ContextPath: "/grafana",
									Hosts: []string{
										"grafana.example.com",
									},
									Metadata: v2.MetadataConfig{
										Annotations: map[string]string{
											"ingress-annotation": "ingress-annotation-value",
										},
										Labels: map[string]string{
											"ingress-label": "ingress-label-value",
										},
									},
									TLS: v1.NewHelmValues(map[string]interface{}{
										"termination": "reencrypt",
									}),
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"ingress": map[string]interface{}{
					"enabled":     true,
					"contextPath": "/grafana",
					"annotations": map[string]interface{}{
						"ingress-annotation": "ingress-annotation-value",
					},
					"labels": map[string]interface{}{
						"ingress-label": "ingress-label-value",
					},
					"hosts": []interface{}{
						"grafana.example.com",
					},
					"tls": map[string]interface{}{
						"termination": "reencrypt",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.service.nodeport." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Service: v2.ComponentServiceConfig{
								NodePort: &grafanaTestNodePort,
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"service": map[string]interface{}{
					"nodePort": map[string]interface{}{
						"enabled": true,
						"port":    12345,
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.runtime.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Runtime: &v2.ComponentRuntimeConfig{},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"autoscaleEnabled": false,
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.runtime.basic." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Runtime: &v2.ComponentRuntimeConfig{
								Deployment: v2.DeploymentRuntimeConfig{
									Replicas: &replicaCount2,
									Strategy: &appsv1.DeploymentStrategy{
										RollingUpdate: &appsv1.RollingUpdateDeployment{
											MaxSurge:       &intStrInt1,
											MaxUnavailable: &intStr25Percent,
										},
									},
								},
								Pod: v2.PodRuntimeConfig{
									CommonPodRuntimeConfig: v2.CommonPodRuntimeConfig{
										NodeSelector: map[string]string{
											"node-label": "node-value",
										},
										PriorityClassName: "normal",
										Tolerations: []corev1.Toleration{
											{
												Key:      "bad-node",
												Operator: corev1.TolerationOpExists,
												Effect:   corev1.TaintEffectNoExecute,
											},
											{
												Key:      "istio",
												Operator: corev1.TolerationOpEqual,
												Value:    "disabled",
												Effect:   corev1.TaintEffectNoSchedule,
											},
										},
									},
									Affinity: &v2.Affinity{
										PodAntiAffinity: v2.PodAntiAffinity{
											PreferredDuringScheduling: []v2.PodAntiAffinityTerm{
												{
													LabelSelectorRequirement: metav1.LabelSelectorRequirement{
														Key:      "istio",
														Operator: metav1.LabelSelectorOpIn,
														Values: []string{
															"control-plane",
														},
													},
												},
											},
											RequiredDuringScheduling: []v2.PodAntiAffinityTerm{
												{
													LabelSelectorRequirement: metav1.LabelSelectorRequirement{
														Key:      "istio",
														Operator: metav1.LabelSelectorOpIn,
														Values: []string{
															"ingressgateway",
														},
													},
												},
											},
										},
									},
									Metadata: v2.MetadataConfig{
										Annotations: map[string]string{
											"some-pod-annotation": "pod-annotation-value",
										},
										Labels: map[string]string{
											"some-pod-label": "pod-label-value",
										},
									},
									Containers: map[string]v2.ContainerConfig{
										"grafana": {
											CommonContainerConfig: v2.CommonContainerConfig{
												ImageRegistry:   "custom-registry",
												ImageTag:        "test",
												ImagePullPolicy: "Always",
												ImagePullSecrets: []corev1.LocalObjectReference{
													{
														Name: "pull-secret",
													},
												},
												Resources: &corev1.ResourceRequirements{
													Limits: corev1.ResourceList{
														corev1.ResourceCPU:    resource.MustParse("100m"),
														corev1.ResourceMemory: resource.MustParse("128Mi"),
													},
													Requests: corev1.ResourceList{
														corev1.ResourceCPU:    resource.MustParse("10m"),
														corev1.ResourceMemory: resource.MustParse("64Mi"),
													},
												},
											},
											Image: "custom-grafana",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"autoscaleEnabled":      false,
				"replicaCount":          2,
				"rollingMaxSurge":       1,
				"rollingMaxUnavailable": "25%",
				"nodeSelector": map[string]interface{}{
					"node-label": "node-value",
				},
				"priorityClassName": "normal",
				"tolerations": []interface{}{
					map[string]interface{}{
						"effect":   "NoExecute",
						"key":      "bad-node",
						"operator": "Exists",
					},
					map[string]interface{}{
						"effect":   "NoSchedule",
						"key":      "istio",
						"operator": "Equal",
						"value":    "disabled",
					},
				},
				"podAntiAffinityTermLabelSelector": []interface{}{
					map[string]interface{}{
						"key":         "istio",
						"operator":    "In",
						"topologyKey": "",
						"values":      "control-plane",
					},
				},
				"podAntiAffinityLabelSelector": []interface{}{
					map[string]interface{}{
						"key":         "istio",
						"operator":    "In",
						"topologyKey": "",
						"values":      "ingressgateway",
					},
				},
				"podAnnotations": map[string]interface{}{
					"some-pod-annotation": "pod-annotation-value",
				},
				"podLabels": map[string]interface{}{
					"some-pod-label": "pod-label-value",
				},
				"hub":             "custom-registry",
				"image":           "custom-grafana",
				"tag":             "test",
				"imagePullPolicy": "Always",
				"imagePullSecrets": []interface{}{
					"pull-secret",
				},
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "100m",
						"memory": "128Mi",
					},
					"requests": map[string]interface{}{
						"cpu":    "10m",
						"memory": "64Mi",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
	{
		name: "install.runtime.autoscale." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Visualization: v2.VisualizationAddonsConfig{
					Grafana: &v2.GrafanaAddonConfig{
						Install: &v2.GrafanaInstallConfig{
							Runtime: &v2.ComponentRuntimeConfig{
								Deployment: v2.DeploymentRuntimeConfig{
									Replicas: &replicaCount2,
									AutoScaling: &v2.AutoScalerConfig{
										MaxReplicas:                    &replicaCount5,
										MinReplicas:                    &replicaCount1,
										TargetCPUUtilizationPercentage: &cpuUtilization80,
									},
									Strategy: &appsv1.DeploymentStrategy{
										RollingUpdate: &appsv1.RollingUpdateDeployment{
											MaxSurge:       &intStr25Percent,
											MaxUnavailable: &intStrInt1,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"grafana": map[string]interface{}{
				"autoscaleEnabled": true,
				"autoscaleMax":     5,
				"autoscaleMin":     1,
				"cpu": map[string]interface{}{
					"targetAverageUtilization": 80,
				},
				"replicaCount":          2,
				"rollingMaxSurge":       "25%",
				"rollingMaxUnavailable": 1,
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"useMCP": true,
				"multiCluster": map[string]interface{}{
					"enabled": false,
				},
				"meshExpansion": map[string]interface{}{
					"enabled": false,
					"useILB":  false,
				},
			},
			"istio_cni": map[string]interface{}{
				"enabled": true,
			},
		}),
	},
}

func TestGrafanaConversionFromV2(t *testing.T) {
	for _, tc := range grafanaTestCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.DeepCopy()
			helmValues := v1.NewHelmValues(make(map[string]interface{}))
			if err := populateAddonsValues(specCopy, helmValues.GetContent()); err != nil {
				t.Fatalf("error converting to values: %s", err)
			}
			if !reflect.DeepEqual(tc.isolatedIstio.DeepCopy(), helmValues.DeepCopy()) {
				t.Errorf("unexpected output converting v2 to values:\n\texpected:\n%#v\n\tgot:\n%#v", tc.isolatedIstio.GetContent(), helmValues.GetContent())
			}
			specv2 := &v2.ControlPlaneSpec{}
			// use expected values
			helmValues = tc.isolatedIstio.DeepCopy()
			mergeMaps(tc.completeIstio.DeepCopy().GetContent(), helmValues.GetContent())
			if err := populateAddonsConfig(helmValues.DeepCopy(), specv2); err != nil {
				t.Fatalf("error converting from values: %s", err)
			}
			if !reflect.DeepEqual(tc.spec.Addons, specv2.Addons) {
				expected, _ := yaml.Marshal(tc.spec.Addons)
				got, _ := yaml.Marshal(specv2.Addons)
				t.Errorf("unexpected output converting values back to v2:\n\texpected:\n%s\n\tgot:\n%s", string(expected), string(got))
			}
		})
	}
}