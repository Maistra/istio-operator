package conversion

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/versions"
)

var gatewaysTestCases = []conversionTestCase{
	{
		name: "nil." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version:  versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "enablement." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				Enablement: v2.Enablement{
					Enabled: &featureEnabled,
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"enabled": true,
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.disabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     false,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.enabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.ior_enabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				OpenShiftRoute: &v2.OpenShiftRouteConfig{
					Enablement: v2.Enablement{
						Enabled: &featureEnabled,
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"ior_enabled": true,
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.service.basic." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Service: v2.GatewayServiceConfig{
								ServiceSpec: corev1.ServiceSpec{
									Ports: []corev1.ServicePort{
										{
											Name:       "http",
											Port:       80,
											TargetPort: intstr.FromInt(8080),
										},
										{
											Name:       "https",
											Port:       443,
											TargetPort: intstr.FromInt(8443),
										},
										{
											Name:       "tcp-custom",
											Port:       24443,
											TargetPort: intstr.FromString("tcp-custom"),
										},
									},
									Type: corev1.ServiceTypeClusterIP,
								},
								Metadata: &v2.MetadataConfig{
									Labels: map[string]string{
										"extra-label": "label-value",
									},
									Annotations: map[string]string{
										"some-annotation": "not-used-in-charts",
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"labels": map[string]interface{}{
						"extra-label": "label-value",
					},
					"serviceAnnotations": map[string]interface{}{
						"some-annotation": "not-used-in-charts",
					},
					"ports": []interface{}{
						map[string]interface{}{
							"name":       "http",
							"port":       80,
							"targetPort": 8080,
						},
						map[string]interface{}{
							"name":       "https",
							"port":       443,
							"targetPort": 8443,
						},
						map[string]interface{}{
							"name":       "tcp-custom",
							"port":       24443,
							"targetPort": "tcp-custom",
						},
					},
					"type": "ClusterIP",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.service.loadbalancer." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Service: v2.GatewayServiceConfig{
								ServiceSpec: corev1.ServiceSpec{
									Ports: []corev1.ServicePort{
										{
											Name:       "http",
											Port:       80,
											TargetPort: intstr.FromInt(8080),
										},
										{
											Name:       "https",
											Port:       443,
											TargetPort: intstr.FromInt(8443),
										},
										{
											Name:       "tcp-custom",
											Port:       24443,
											TargetPort: intstr.FromString("tcp-custom"),
										},
									},
									Type:                  corev1.ServiceTypeLoadBalancer,
									ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
									ExternalIPs: []string{
										"192.168.1.21",
									},
									LoadBalancerIP: "10.30.29.28",
									LoadBalancerSourceRanges: []string{
										"10.1.2.0/24",
									},
								},
								Metadata: &v2.MetadataConfig{
									Labels: map[string]string{
										"extra-label": "label-value",
									},
									Annotations: map[string]string{
										"some-annotation": "not-used-in-charts",
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"labels": map[string]interface{}{
						"extra-label": "label-value",
					},
					"serviceAnnotations": map[string]interface{}{
						"some-annotation": "not-used-in-charts",
					},
					"ports": []interface{}{
						map[string]interface{}{
							"name":       "http",
							"port":       80,
							"targetPort": 8080,
						},
						map[string]interface{}{
							"name":       "https",
							"port":       443,
							"targetPort": 8443,
						},
						map[string]interface{}{
							"name":       "tcp-custom",
							"port":       24443,
							"targetPort": "tcp-custom",
						},
					},
					"type": "LoadBalancer",
					"externalIPs": []interface{}{
						"192.168.1.21",
					},
					"externalTrafficPolicy": "Cluster",
					"loadBalancerIP":        "10.30.29.28",
					"loadBalancerSourceRanges": []interface{}{
						"10.1.2.0/24",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.service.nodeport." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Service: v2.GatewayServiceConfig{
								ServiceSpec: corev1.ServiceSpec{
									Ports: []corev1.ServicePort{
										{
											Name:       "http",
											Port:       80,
											TargetPort: intstr.FromInt(8080),
										},
										{
											Name:       "https",
											Port:       443,
											TargetPort: intstr.FromInt(8443),
										},
										{
											Name:       "tcp-custom",
											Port:       24443,
											NodePort:   34443,
											TargetPort: intstr.FromString("tcp-custom"),
										},
									},
									Type: corev1.ServiceTypeNodePort,
								},
								Metadata: &v2.MetadataConfig{
									Labels: map[string]string{
										"extra-label": "label-value",
									},
									Annotations: map[string]string{
										"some-annotation": "not-used-in-charts",
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"labels": map[string]interface{}{
						"extra-label": "label-value",
					},
					"serviceAnnotations": map[string]interface{}{
						"some-annotation": "not-used-in-charts",
					},
					"ports": []interface{}{
						map[string]interface{}{
							"name":       "http",
							"port":       80,
							"targetPort": 8080,
						},
						map[string]interface{}{
							"name":       "https",
							"port":       443,
							"targetPort": 8443,
						},
						map[string]interface{}{
							"name":       "tcp-custom",
							"port":       24443,
							"nodePort":   34443,
							"targetPort": "tcp-custom",
						},
					},
					"type": "NodePort",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.basicfields." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Namespace:  "custom-namespace",
							RouterMode: v2.RouterModeTypeSNIDNAT,
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"namespace":   "custom-namespace",
					"env": map[string]interface{}{
						"ISTIO_META_ROUTER_MODE": "sni-dnat",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.volumes." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Volumes: []v2.VolumeConfig{
								{
									Volume: v2.GatewayVolume{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "some-secret",
										},
									},
									Mount: corev1.VolumeMount{
										Name:      "secret-mount",
										MountPath: "/my/secret",
									},
								},
								{
									Volume: v2.GatewayVolume{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "some-config-map",
											},
										},
									},
									Mount: corev1.VolumeMount{
										Name:      "config-map-mount",
										MountPath: "/my/configmap",
									},
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"secretVolumes": []interface{}{
						map[string]interface{}{
							"name":       "secret-mount",
							"secretName": "some-secret",
							"mountPath":  "/my/secret",
						},
					},
					"configVolumes": []interface{}{
						map[string]interface{}{
							"name":          "config-map-mount",
							"configMapName": "some-config-map",
							"mountPath":     "/my/configmap",
						},
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.runtime.basic." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Runtime: &v2.ComponentRuntimeConfig{
								Deployment: &v2.DeploymentRuntimeConfig{
									Replicas: &replicaCount2,
									Strategy: &appsv1.DeploymentStrategy{
										RollingUpdate: &appsv1.RollingUpdateDeployment{
											MaxSurge:       &intStrInt1,
											MaxUnavailable: &intStr25Percent,
										},
									},
								},
								Pod: &v2.PodRuntimeConfig{
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
									Metadata: &v2.MetadataConfig{
										Annotations: map[string]string{
											"some-pod-annotation": "pod-annotation-value",
										},
										Labels: map[string]string{
											"some-pod-label": "pod-label-value",
										},
									},
								},
								Container: &v2.ContainerConfig{
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
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":               true,
					"name":                  "istio-ingressgateway",
					"gatewayType":           "ingress",
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
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.runtime.autoscale." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Runtime: &v2.ComponentRuntimeConfig{
								Deployment: &v2.DeploymentRuntimeConfig{
									Replicas: &replicaCount2,
									AutoScaling: &v2.AutoScalerConfig{
										Enablement:                     v2.Enablement{Enabled: &featureEnabled},
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
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":          true,
					"name":             "istio-ingressgateway",
					"gatewayType":      "ingress",
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
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.sds.simple.disabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
						SDS: &v2.SecretDiscoveryService{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"sds": map[string]interface{}{
						"enabled": false,
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.sds.simple.enabled" + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
						SDS: &v2.SecretDiscoveryService{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"sds": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.sds.runtime.enabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
						SDS: &v2.SecretDiscoveryService{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
							Runtime: &v2.ContainerConfig{
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
								Image: "custom-sds-image",
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"sds": map[string]interface{}{
						"enabled":         true,
						"hub":             "custom-registry",
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
						"image": "custom-sds-image",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.sds.runtime.disabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{
					IngressGatewayConfig: v2.IngressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
						SDS: &v2.SecretDiscoveryService{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
							Runtime: &v2.ContainerConfig{
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
								Image: "custom-sds-image",
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
					"sds": map[string]interface{}{
						"enabled":         false,
						"hub":             "custom-registry",
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
						"image": "custom-sds-image",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "additional.ingress." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				IngressGateways: map[string]*v2.IngressGatewayConfig{
					"extra-ingress-enabled": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
					},
					"extra-ingress-disabled": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
						},
					},
					"extra-ingress-runtime": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
							Runtime: &v2.ComponentRuntimeConfig{
								Deployment: &v2.DeploymentRuntimeConfig{},
								Container: &v2.ContainerConfig{
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
									Image: "custom-proxy-image",
								},
							},
						},
					},
					"extra-ingress-sds": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
						SDS: &v2.SecretDiscoveryService{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
					},
					"extra-ingress-sds-runtime": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
						},
						SDS: &v2.SecretDiscoveryService{
							Runtime: &v2.ContainerConfig{
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
								Image: "custom-sds-image",
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"extra-ingress-enabled": map[string]interface{}{
					"enabled":     true,
					"name":        "extra-ingress-enabled",
					"gatewayType": "ingress",
				},
				"extra-ingress-disabled": map[string]interface{}{
					"enabled":     false,
					"name":        "extra-ingress-disabled",
					"gatewayType": "ingress",
				},
				"extra-ingress-runtime": map[string]interface{}{
					"enabled":         false,
					"name":            "extra-ingress-runtime",
					"gatewayType":     "ingress",
					"hub":             "custom-registry",
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
					"image": "custom-proxy-image",
				},
				"extra-ingress-sds": map[string]interface{}{
					"enabled":     true,
					"name":        "extra-ingress-sds",
					"gatewayType": "ingress",
					"sds": map[string]interface{}{
						"enabled": true,
					},
				},
				"extra-ingress-sds-runtime": map[string]interface{}{
					"enabled":     false,
					"name":        "extra-ingress-sds-runtime",
					"gatewayType": "ingress",
					"sds": map[string]interface{}{
						"hub":             "custom-registry",
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
						"image": "custom-sds-image",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "egress.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterEgress: &v2.EgressGatewayConfig{
					// XXX: round-trip tests fail without this
					RequestedNetworkView: []string{},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-egressgateway": map[string]interface{}{
					"name":        "istio-egressgateway",
					"gatewayType": "egress",
					"env": map[string]interface{}{
						"ISTIO_META_REQUESTED_NETWORK_VIEW": "",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "egress.disabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterEgress: &v2.EgressGatewayConfig{
					GatewayConfig: v2.GatewayConfig{
						Enablement: v2.Enablement{
							Enabled: &featureDisabled,
						},
					},
					// XXX: round-trip tests fail without this
					RequestedNetworkView: []string{},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-egressgateway": map[string]interface{}{
					"enabled":     false,
					"name":        "istio-egressgateway",
					"gatewayType": "egress",
					"env": map[string]interface{}{
						"ISTIO_META_REQUESTED_NETWORK_VIEW": "",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "egress.enabled." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterEgress: &v2.EgressGatewayConfig{
					GatewayConfig: v2.GatewayConfig{
						Enablement: v2.Enablement{
							Enabled: &featureEnabled,
						},
					},
					// XXX: round-trip tests fail without this
					RequestedNetworkView: []string{},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-egressgateway": map[string]interface{}{
					"enabled":     true,
					"name":        "istio-egressgateway",
					"gatewayType": "egress",
					"env": map[string]interface{}{
						"ISTIO_META_REQUESTED_NETWORK_VIEW": "",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "additional.egress." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				EgressGateways: map[string]*v2.EgressGatewayConfig{
					"extra-egress-enabled": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
					},
					"extra-egress-disabled": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
						},
						RequestedNetworkView: []string{},
					},
					"extra-egress-runtime": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureDisabled,
							},
							Runtime: &v2.ComponentRuntimeConfig{
								Deployment: &v2.DeploymentRuntimeConfig{},
								Container: &v2.ContainerConfig{
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
									Image: "custom-proxy-image",
								},
							},
						},
						RequestedNetworkView: []string{},
					},
					"extra-egress-network": {
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &featureEnabled,
							},
						},
						RequestedNetworkView: []string{
							"external",
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"extra-egress-enabled": map[string]interface{}{
					"enabled":     true,
					"name":        "extra-egress-enabled",
					"gatewayType": "egress",
				},
				"extra-egress-disabled": map[string]interface{}{
					"enabled":     false,
					"name":        "extra-egress-disabled",
					"gatewayType": "egress",
					"env": map[string]interface{}{
						"ISTIO_META_REQUESTED_NETWORK_VIEW": "",
					},
				},
				"extra-egress-runtime": map[string]interface{}{
					"enabled":     false,
					"name":        "extra-egress-runtime",
					"gatewayType": "egress",
					"env": map[string]interface{}{
						"ISTIO_META_REQUESTED_NETWORK_VIEW": "",
					},
					"hub":             "custom-registry",
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
					"image": "custom-proxy-image",
				},
				"extra-egress-network": map[string]interface{}{
					"enabled":     true,
					"name":        "extra-egress-network",
					"gatewayType": "egress",
					"env": map[string]interface{}{
						"ISTIO_META_REQUESTED_NETWORK_VIEW": "external",
					},
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "ingress.additionalContainers." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Gateways: &v2.GatewaysConfig{
				ClusterIngress: &v2.ClusterIngressGatewayConfig{},
			},
			TechPreview: v1.NewHelmValues(map[string]interface{}{
				"gateways": map[string]interface{}{
					"istio-ingressgateway": map[string]interface{}{
						"additionalContainers": []interface{}{
							map[string]interface{}{
								"name":            "custom-container",
								"image":           "custom-image",
								"imagePullPolicy": "Always",
							},
						},
					},
				},
			}),
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"name":        "istio-ingressgateway",
					"gatewayType": "ingress",
				},
			},
		}),
		completeIstio: v1.NewHelmValues(map[string]interface{}{
			"global": map[string]interface{}{
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
			"gateways": map[string]interface{}{
				"istio-ingressgateway": map[string]interface{}{
					"additionalContainers": []interface{}{
						map[string]interface{}{
							"name":            "custom-container",
							"image":           "custom-image",
							"imagePullPolicy": "Always",
						},
					},
				},
			},
		}),
	},
}

func TestGatewaysConversionFromV2(t *testing.T) {
	for _, tc := range gatewaysTestCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.DeepCopy()
			helmValues := v1.NewHelmValues(make(map[string]interface{}))
			if err := populateGatewaysValues(specCopy, helmValues.GetContent()); err != nil {
				t.Fatalf("error converting to values: %s", err)
			}
			if !reflect.DeepEqual(tc.isolatedIstio.DeepCopy(), helmValues.DeepCopy()) {
				t.Errorf("unexpected output converting v2 to values:\n\texpected:\n%#v\n\tgot:\n%#v", tc.isolatedIstio.GetContent(), helmValues.GetContent())
			}
			specv2 := &v2.ControlPlaneSpec{}
			// use expected values
			helmValues = tc.isolatedIstio.DeepCopy()
			mergeMaps(tc.completeIstio.DeepCopy().GetContent(), helmValues.GetContent())
			if err := populateGatewaysConfig(helmValues.DeepCopy(), specv2); err != nil {
				t.Fatalf("error converting from values: %s", err)
			}
			assertEquals(t, tc.spec.Gateways, specv2.Gateways)
		})
	}
}
