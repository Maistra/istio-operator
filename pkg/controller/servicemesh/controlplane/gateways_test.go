package controlplane

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/common"
	. "github.com/maistra/istio-operator/pkg/controller/common/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"
)

func TestAdditionalIngressGatewayInstall(t *testing.T) {
	enabled := true
	additionalGatewayName := "additional-gateway"
	appNamespace := "app-namespace"
	testCases := []IntegrationTestCase{
		{
			name: "no-namespace",
			smcp: New20SMCPResource(controlPlaneName, controlPlaneNamespace, &v2.ControlPlaneSpec{
				Gateways: &v2.GatewaysConfig{
					ClusterIngress: &v2.ClusterIngressGatewayConfig{
						IngressGatewayConfig: v2.IngressGatewayConfig{
							GatewayConfig: v2.GatewayConfig{
								Enablement: v2.Enablement{
									Enabled: &enabled,
								},
							},
						},
					},
					IngressGateways: map[string]*v2.IngressGatewayConfig{
						additionalGatewayName: {
							GatewayConfig: v2.GatewayConfig{
								Enablement: v2.Enablement{
									Enabled: &enabled,
								},
							},
						},
					},
				},
			}),
			create: IntegrationTestValidation{
				Verifier: VerifyActions(
					Verify("create").On("deployments").Named("istio-ingressgateway").In(controlPlaneNamespace).Passes(ExpectedDefaultLabelGatewayCreate("istio-ingressgateway."+controlPlaneNamespace)),
					Verify("create").On("deployments").Named(additionalGatewayName).In(controlPlaneNamespace).Passes(ExpectedDefaultLabelGatewayCreate(additionalGatewayName+"."+controlPlaneNamespace)),
				),
				Assertions: ActionAssertions{},
			},
			delete: IntegrationTestValidation{
				Assertions: ActionAssertions{
					Assert("delete").On("deployments").Named(additionalGatewayName).In(controlPlaneNamespace).IsSeen(),
				},
			},
		},
		{
			name: "cp-namespace",
			smcp: New20SMCPResource(controlPlaneName, controlPlaneNamespace, &v2.ControlPlaneSpec{
				Gateways: &v2.GatewaysConfig{
					IngressGateways: map[string]*v2.IngressGatewayConfig{
						additionalGatewayName: {
							GatewayConfig: v2.GatewayConfig{
								Enablement: v2.Enablement{
									Enabled: &enabled,
								},
								Namespace: controlPlaneNamespace,
							},
						},
					},
				},
			}),
			create: IntegrationTestValidation{
				Verifier: ActionVerifier(
					Verify("create").On("deployments").Named(additionalGatewayName).In(controlPlaneNamespace).Passes(ExpectedDefaultLabelGatewayCreate(additionalGatewayName + "." + controlPlaneNamespace)),
				),
				Assertions: ActionAssertions{},
			},
			delete: IntegrationTestValidation{
				Assertions: ActionAssertions{
					Assert("delete").On("deployments").Named(additionalGatewayName).In(controlPlaneNamespace).IsSeen(),
				},
			},
		},
		{
			name: "app-namespace",
			resources: []runtime.Object{
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: appNamespace}},
				&v1.ServiceMeshMemberRoll{
					ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: controlPlaneNamespace},
					Status: v1.ServiceMeshMemberRollStatus{
						ConfiguredMembers: []string{
							appNamespace,
						},
					},
				},
			},
			smcp: New20SMCPResource(controlPlaneName, controlPlaneNamespace, &v2.ControlPlaneSpec{
				Gateways: &v2.GatewaysConfig{
					IngressGateways: map[string]*v2.IngressGatewayConfig{
						additionalGatewayName: {
							GatewayConfig: v2.GatewayConfig{
								Enablement: v2.Enablement{
									Enabled: &enabled,
								},
								Namespace: appNamespace,
							},
						},
					},
				},
			}),
			create: IntegrationTestValidation{
				Verifier: ActionVerifier(
					Verify("create").On("deployments").Named(additionalGatewayName).In(appNamespace).Passes(ExpectedExternalGatewayCreate),
				),
				Assertions: ActionAssertions{},
			},
			delete: IntegrationTestValidation{
				Assertions: ActionAssertions{
					// TODO: MAISTRA-1333 gateways in other namepsaces do not get deleted properly
					//Assert("delete").On("deployments").Named(additionalGatewayName).In(appNamespace).IsSeen(),
				},
			},
		},
	}
	RunSimpleInstallTest(t, testCases)
}

func TestClusterGatewaysOutsideCPNamespace(t *testing.T) {
	enabled := true
	appNamespace := "app-namespace"
	smmr := &v1.ServiceMeshMemberRoll{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: controlPlaneNamespace},
		Spec: v1.ServiceMeshMemberRollSpec{
			Members: []string{appNamespace},
		},
		Status: v1.ServiceMeshMemberRollStatus{
			ConfiguredMembers: []string{appNamespace},
		},
	}

	testCases := []IntegrationTestCase{
		{
			name: "ingress",
			smcp: New20SMCPResource(controlPlaneName, controlPlaneNamespace, &v2.ControlPlaneSpec{
				Gateways: &v2.GatewaysConfig{
					ClusterIngress: &v2.ClusterIngressGatewayConfig{
						IngressGatewayConfig: v2.IngressGatewayConfig{
							GatewayConfig: v2.GatewayConfig{
								Enablement: v2.Enablement{
									Enabled: &enabled,
								},
								Namespace: appNamespace,
							},
						},
					},
				},
			}),
			resources: []runtime.Object{smmr},
			create: IntegrationTestValidation{
				Verifier: VerifyActions(
					Verify("create").On("deployments").Named("istio-ingressgateway").In(appNamespace).Passes(ExpectedExternalGatewayCreate),
				),
				Assertions: ActionAssertions{
					&pausedStatusUpdate{ActionAssertion: Assert("patch").On("servicemeshcontrolplanes/status").In(controlPlaneNamespace).IsNotSeen()},
				},
			},
		},
		{
			name: "egress",
			smcp: New20SMCPResource(controlPlaneName, controlPlaneNamespace, &v2.ControlPlaneSpec{
				Gateways: &v2.GatewaysConfig{
					ClusterEgress: &v2.EgressGatewayConfig{
						GatewayConfig: v2.GatewayConfig{
							Enablement: v2.Enablement{
								Enabled: &enabled,
							},
							Namespace: appNamespace,
						},
					},
				},
			}),
			resources: []runtime.Object{smmr},
			create: IntegrationTestValidation{
				Verifier: VerifyActions(
					Verify("create").On("deployments").Named("istio-egressgateway").In(appNamespace).Passes(ExpectedExternalGatewayCreate),
				),
				Assertions: ActionAssertions{
					&pausedStatusUpdate{ActionAssertion: Assert("patch").On("servicemeshcontrolplanes/status").In(controlPlaneNamespace).IsNotSeen()},
				},
			},
		},
	}
	RunSimpleInstallTest(t, testCases)
}

type pausedStatusUpdate struct {
	ActionAssertion
}

var _ ActionAssertion = (*pausedStatusUpdate)(nil)

func (a *pausedStatusUpdate) React(action clienttesting.Action) (bool, runtime.Object, error) {
	patchAction := action.(clienttesting.PatchAction)
	if strings.Contains(string(patchAction.GetPatch()), "Pausing") {
		return a.ActionAssertion.React(action)
	}
	return false, nil, nil
}

func ExpectedDefaultLabelGatewayCreate(expected string) func(action clienttesting.Action) error {
	return func(action clienttesting.Action) error {
		createAction := action.(clienttesting.CreateAction)
		obj := createAction.GetObject()
		gateway := obj.(*unstructured.Unstructured)
		if val, ok := common.GetLabel(gateway, "maistra.io/gateway"); ok {
			if val != expected {
				return fmt.Errorf("expected maistra.io/gateway label to be %s, got %s", expected, val)
			}
		} else {
			return fmt.Errorf("gateway should have maistra.io/gateway label defined")
		}
		return nil
	}
}

func ExpectedExternalGatewayCreate(action clienttesting.Action) error {
	createAction := action.(clienttesting.CreateAction)
	obj := createAction.GetObject()
	gateway := obj.(*unstructured.Unstructured)
	if len(gateway.GetOwnerReferences()) > 0 {
		return fmt.Errorf("external gateway should not have an owner reference")
	}
	return nil
}
