package bootstrap

import (
	"context"
	"github.com/ghodss/yaml"
	"github.com/maistra/istio-operator/pkg/controller/common"
	"github.com/maistra/istio-operator/pkg/controller/common/cni"
	"github.com/maistra/istio-operator/pkg/controller/versions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path"
	goruntime "runtime"
	"testing"

	"github.com/maistra/istio-operator/pkg/controller/common/test"
	"github.com/maistra/istio-operator/pkg/controller/common/test/assert"
)

func TestCNISupportedVersionRendering(t *testing.T) {
	operatorNamespace := "istio-operator"
	InitializeGlobals(operatorNamespace)()

	var ctx = context.Background()

	testCases := []struct {
		name              string
		supportedVersions []versions.Version
		containerNames    []string
	}{
		{
			name:              "Default Supported Versions",
			supportedVersions: versions.GetSupportedVersions(),
			containerNames:    []string{"install-cni-v1-0", "install-cni-v1-1", "install-cni-v2-0"},
		},
		{
			name:              "v1.0 only",
			supportedVersions: []versions.Version{versions.V1_0},
			containerNames:    []string{"install-cni-v1-0"},
		},
		{
			name:              "v1.1 only",
			supportedVersions: []versions.Version{versions.V1_1},
			containerNames:    []string{"install-cni-v1-1"},
		},
		{
			name:              "v2.0 only",
			supportedVersions: []versions.Version{versions.V2_0},
			containerNames:    []string{"install-cni-v2-0"},
		},
	}

	config := cni.Config{
		Enabled: true,
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl, _ := test.CreateClient()
			renderings, err := internalRenderCNI(ctx, cl, config, tc.supportedVersions)
			assert.Success(err, "internalRenderCNI", t)
			assert.True(renderings != nil, "renderings should not be nil", t)
			cniManifests := renderings["istio_cni"]
			assert.True(cniManifests != nil, "CNI manifests are not present", t)

			var foundDaemonSet bool

			for _, manifest := range cniManifests {
				if manifest.Head.Kind == "DaemonSet" {
					foundDaemonSet = true
					json, err := yaml.YAMLToJSON([]byte(manifest.Content))
					assert.Success(err, "YAMLToJSON", t)
					resource := &unstructured.Unstructured{}
					_, _, err = unstructured.UnstructuredJSONScheme.Decode(json, nil, resource)
					assert.Success(err, "resource decoding", t)

					containers, found, err := unstructured.NestedSlice(resource.UnstructuredContent(), "spec", "template", "spec", "containers")
					assert.True(found, "Could not find containers", t)
					assert.True(len(containers) > 0, "No containers in resource", t)
					names := []string{}
					for _, container := range containers {
						val, ok := container.(map[string]interface{})
						assert.True(ok, "Converting container to map", t)
						names = append(names, val["name"].(string))
					}
					assert.DeepEquals(names, tc.containerNames, "Unexpected container name set", t)
				}
			}
			assert.True(foundDaemonSet, "Daemon Set was not in Manifest list", t)
		})
	}
}

// InitializeGlobals returns a function which initializes global variables used
// by the system under test.  operatorNamespace is the namespace within which
// the operator is installed.
func InitializeGlobals(operatorNamespace string) func() {
	return func() {
		// make sure globals are initialized for testing
		common.Config.OLM.Images.V1_0.CNI = "istio-cni-test-1_0"
		common.Config.OLM.Images.V1_1.CNI = "istio-cni-test-1_1"
		common.Config.OLM.Images.V2_0.CNI = "istio-cni-test-2_0"
		common.Config.OLM.Images.V1_0.ThreeScale = "injected-3scale-v1.0"
		common.Config.OLM.Images.V1_1.ThreeScale = "injected-3scale-v1.1"
		common.Config.OLM.Images.V2_0.ThreeScale = "injected-3scale-v2.0"
		common.Config.OLM.Images.V1_0.Citadel = "injected-citadel-v1.0"
		common.Config.OLM.Images.V1_1.Citadel = "injected-citadel-v1.1"
		common.Config.OLM.Images.V1_0.Galley = "injected-galley-v1.0"
		common.Config.OLM.Images.V1_1.Galley = "injected-galley-v1.1"
		common.Config.OLM.Images.V1_0.Grafana = "injected-grafana-v1.0"
		common.Config.OLM.Images.V1_1.Grafana = "injected-grafana-v1.1"
		common.Config.OLM.Images.V2_0.Grafana = "injected-grafana-v2.0"
		common.Config.OLM.Images.V1_0.Mixer = "injected-mixer-v1.0"
		common.Config.OLM.Images.V1_1.Mixer = "injected-mixer-v1.1"
		common.Config.OLM.Images.V2_0.Mixer = "injected-mixer-v2.0"
		common.Config.OLM.Images.V1_0.Pilot = "injected-pilot-v1.0"
		common.Config.OLM.Images.V1_1.Pilot = "injected-pilot-v1.1"
		common.Config.OLM.Images.V2_0.Pilot = "injected-pilot-v2.0"
		common.Config.OLM.Images.V1_0.Prometheus = "injected-prometheus-v1.0"
		common.Config.OLM.Images.V1_1.Prometheus = "injected-prometheus-v1.1"
		common.Config.OLM.Images.V2_0.Prometheus = "injected-prometheus-v2.0"
		common.Config.OLM.Images.V1_0.ProxyInit = "injected-proxy-init-v1.0"
		common.Config.OLM.Images.V1_1.ProxyInit = "injected-proxy-init-v1.1"
		common.Config.OLM.Images.V2_0.ProxyInit = "injected-proxy-init-v2.0"
		common.Config.OLM.Images.V1_0.ProxyV2 = "injected-proxyv2-v1.0"
		common.Config.OLM.Images.V1_1.ProxyV2 = "injected-proxyv2-v1.1"
		common.Config.OLM.Images.V2_0.ProxyV2 = "injected-proxyv2-v2.0"
		common.Config.OLM.Images.V1_0.SidecarInjector = "injected-sidecar-injector-v1.0"
		common.Config.OLM.Images.V1_1.SidecarInjector = "injected-sidecar-injector-v1.1"
		common.Config.OLM.Images.V1_1.IOR = "injected-ior-v1.1"
		common.Config.OLM.Images.V2_0.WASMCacher = "injected-wasm-cacher-v2.0"
		os.Setenv("POD_NAMESPACE", operatorNamespace)
		common.GetOperatorNamespace()
		if _, filename, _, ok := goruntime.Caller(0); ok {
			common.Config.Rendering.ResourceDir = path.Join(path.Dir(filename), "../../resources")
			common.Config.Rendering.ChartsDir = path.Join(common.Config.Rendering.ResourceDir, "helm")
			common.Config.Rendering.DefaultTemplatesDir = path.Join(common.Config.Rendering.ResourceDir, "smcp-templates")
		} else {
			panic("could not initialize common.ResourceDir")
		}
	}
}
