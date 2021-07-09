package conversion

import (
	"reflect"
	"testing"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/versions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var (
	kialiTestNodePort = int32(12345)
)

var kialiTestCases = []conversionTestCase{
	{
		name: "nil." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: nil,
			},
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
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{},
			},
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
		name: "simple." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Enablement: v2.Enablement{
						Enabled: &featureEnabled,
					},
					Name: "my-kiali",
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"enabled":      true,
				"resourceName": "my-kiali",
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
		name: "install.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name:    "my-kiali",
					Install: &v2.KialiInstallConfig{},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
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
		name: "install.config.simple." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Dashboard: &v2.KialiDashboardConfig{
							EnableGrafana:    &featureEnabled,
							EnablePrometheus: &featureEnabled,
							EnableTracing:    &featureDisabled,
							ViewOnly:         &featureEnabled,
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
				"dashboard": map[string]interface{}{
					"enableGrafana":    true,
					"enablePrometheus": true,
					"enableTracing":    false,
					"viewOnlyMode":     true,
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
		name: "install.service.misc." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Service: &v2.ComponentServiceConfig{
							Metadata: &v2.MetadataConfig{
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
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
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
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "install.service.ingress.defaults." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Service: &v2.ComponentServiceConfig{
							Ingress: &v2.ComponentIngressConfig{},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
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
		name: "install.service.ingress.full." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Service: &v2.ComponentServiceConfig{
							Ingress: &v2.ComponentIngressConfig{
								Enablement: v2.Enablement{
									Enabled: &featureEnabled,
								},
								ContextPath: "/kiali",
								Hosts: []string{
									"kiali.example.com",
								},
								Metadata: &v2.MetadataConfig{
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
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
				"contextPath":  "/kiali",
				"ingress": map[string]interface{}{
					"enabled":     true,
					"contextPath": "/kiali",
					"annotations": map[string]interface{}{
						"ingress-annotation": "ingress-annotation-value",
					},
					"labels": map[string]interface{}{
						"ingress-label": "ingress-label-value",
					},
					"hosts": []interface{}{
						"kiali.example.com",
					},
					"tls": map[string]interface{}{
						"termination": "reencrypt",
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
		name: "install.service.nodeport." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Service: &v2.ComponentServiceConfig{
							NodePort: &kialiTestNodePort,
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
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
				"multiCluster":  globalMultiClusterDefaults,
				"meshExpansion": globalMeshExpansionDefaults,
			},
		}),
	},
	{
		name: "deployment.resources." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Deployment: &v2.KialiDeploymentConfig{
							Resources: &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
				"deployment_resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    "10m",
						"memory": "128Mi",
					},
					"limits": map[string]interface{}{
						"cpu":    "100m",
						"memory": "1Gi",
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
		name: "deployment.affinity." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Deployment: &v2.KialiDeploymentConfig{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "key1",
														Operator: "op1",
														Values:   []string{"value11", "value12"},
													},
												},
											},
										},
									},
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
										{
											Weight: 1,
											Preference: corev1.NodeSelectorTerm{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "key2",
														Operator: "op2",
														Values:   []string{"value21", "value22"},
													},
												},
											},
										},
									},
								},
								PodAffinity: &corev1.PodAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											LabelSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"fookey": "foovalue",
												},
											},
											Namespaces:  []string{"ns1", "ns2"},
											TopologyKey: "my-topology-key",
										},
									},
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 2,
											PodAffinityTerm: corev1.PodAffinityTerm{
												LabelSelector: &metav1.LabelSelector{
													MatchLabels: map[string]string{
														"barkey": "barvalue",
													},
												},
												Namespaces:  []string{"ns3", "ns4"},
												TopologyKey: "my-topology-key2",
											},
										},
									},
								},
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
										{
											LabelSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"bazkey": "bazvalue",
												},
											},
											Namespaces:  []string{"ns5", "ns6"},
											TopologyKey: "my-topology-key3",
										},
									},
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 3,
											PodAffinityTerm: corev1.PodAffinityTerm{
												LabelSelector: &metav1.LabelSelector{
													MatchLabels: map[string]string{
														"quxkey": "quxvalue",
													},
												},
												Namespaces:  []string{"ns7", "ns8"},
												TopologyKey: "my-topology-key4",
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
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
				"deployment_affinity": map[string]interface{}{
					"node": map[string]interface{}{
						"requiredDuringSchedulingIgnoredDuringExecution": map[string]interface{}{
							"nodeSelectorTerms": []interface{}{
								map[string]interface{}{
									"matchFields": []interface{}{
										map[string]interface{}{
											"key":      "key1",
											"operator": "op1",
											"values":   []string{"value11", "value12"},
										},
									},
								},
							},
						},
						"preferredDuringSchedulingIgnoredDuringExecution": []interface{}{
							map[string]interface{}{
								"weight": 1,
								"preference": map[string]interface{}{
									"matchFields": []interface{}{
										map[string]interface{}{
											"key":      "key2",
											"operator": "op2",
											"values":   []string{"value21", "value22"},
										},
									},
								},
							},
						},
					},
					"pod": map[string]interface{}{
						"requiredDuringSchedulingIgnoredDuringExecution": []interface{}{
							map[string]interface{}{
								"labelSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"fookey": "foovalue",
									},
								},
								"namespaces":  []string{"ns1", "ns2"},
								"topologyKey": "my-topology-key",
							},
						},
						"preferredDuringSchedulingIgnoredDuringExecution": []interface{}{
							map[string]interface{}{
								"weight": 2,
								"podAffinityTerm": map[string]interface{}{
									"labelSelector": map[string]interface{}{
										"matchLabels": map[string]interface{}{
											"barkey": "barvalue",
										},
									},
									"namespaces":  []string{"ns3", "ns4"},
									"topologyKey": "my-topology-key2",
								},
							},
						},
					},
					"pod_anti": map[string]interface{}{
						"requiredDuringSchedulingIgnoredDuringExecution": []interface{}{
							map[string]interface{}{
								"labelSelector": map[string]interface{}{
									"matchLabels": map[string]interface{}{
										"bazkey": "bazvalue",
									},
								},
								"namespaces":  []string{"ns5", "ns6"},
								"topologyKey": "my-topology-key3",
							},
						},
						"preferredDuringSchedulingIgnoredDuringExecution": []interface{}{
							map[string]interface{}{
								"weight": 3,
								"podAffinityTerm": map[string]interface{}{
									"labelSelector": map[string]interface{}{
										"matchLabels": map[string]interface{}{
											"quxkey": "quxvalue",
										},
									},
									"namespaces":  []string{"ns7", "ns8"},
									"topologyKey": "my-topology-key4",
								},
							},
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
		name: "deployment.nodeSelector." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Deployment: &v2.KialiDeploymentConfig{
							NodeSelector: map[string]string{
								"fookey": "foovalue",
								"barkey": "barvalue",
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
				"deployment_nodeSelector": map[string]interface{}{
					"fookey": "foovalue",
					"barkey": "barvalue",
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
		name: "deployment.tolerations." + versions.V2_0.String(),
		spec: &v2.ControlPlaneSpec{
			Version: versions.V2_0.String(),
			Addons: &v2.AddonsConfig{
				Kiali: &v2.KialiAddonConfig{
					Name: "my-kiali",
					Install: &v2.KialiInstallConfig{
						Deployment: &v2.KialiDeploymentConfig{
							Tolerations: []corev1.Toleration{
								{
									Key:               "key1",
									Operator:          "op1",
									Value:             "value1",
									Effect:            "effect1",
									TolerationSeconds: pointer.Int64Ptr(1),
								},
								{
									Key:               "key2",
									Operator:          "op2",
									Value:             "value2",
									Effect:            "effect2",
									TolerationSeconds: pointer.Int64Ptr(2),
								},
							},
						},
					},
				},
			},
		},
		isolatedIstio: v1.NewHelmValues(map[string]interface{}{
			"kiali": map[string]interface{}{
				"resourceName": "my-kiali",
				"deployment_tolerations": []interface{}{
					map[string]interface{}{
						"key":               "key1",
						"operator":          "op1",
						"value":             "value1",
						"effect":            "effect1",
						"tolerationSeconds": 1,
					},
					map[string]interface{}{
						"key":               "key2",
						"operator":          "op2",
						"value":             "value2",
						"effect":            "effect2",
						"tolerationSeconds": 2,
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
}

func TestKialiConversionFromV2(t *testing.T) {
	for _, tc := range kialiTestCases {
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
			assertEquals(t, tc.spec.Addons, specv2.Addons)
		})
	}
}
