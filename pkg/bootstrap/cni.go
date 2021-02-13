package bootstrap

import (
	"context"
	"k8s.io/helm/pkg/manifest"
	"path"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/maistra/istio-operator/pkg/controller/common"
	"github.com/maistra/istio-operator/pkg/controller/common/cni"
	"github.com/maistra/istio-operator/pkg/controller/common/helm"
	"github.com/maistra/istio-operator/pkg/controller/versions"
)

// InstallCNI makes sure all Istio CNI resources have been created.  CRDs are located from
// files in controller.HelmDir/istio-init/files
func InstallCNI(ctx context.Context, cl client.Client, config cni.Config) error {
	// we should run through this each reconcile to make sure it's there
	return internalInstallCNI(ctx, cl, config)
}

func internalInstallCNI(ctx context.Context, cl client.Client, config cni.Config) error {
	renderings, err := internalRenderCNI(ctx, cl, config, versions.GetSupportedVersions())
	if err != nil {
		return err
	}
	return internalProcessManifests(ctx, cl, renderings["istio_cni"])
}

func internalRenderCNI(ctx context.Context, cl client.Client, config cni.Config, supportedVersions []versions.Version) (renderings map[string][]manifest.Manifest, err error) {
	log := common.LogFromContext(ctx)
	log.Info("ensuring Istio CNI has been installed")

	operatorNamespace := common.GetOperatorNamespace()

	log.Info("rendering Istio CNI chart")

	values := make(map[string]interface{})
	values["enabled"] = config.Enabled
	values["image_v1_0"] = common.Config.OLM.Images.V1_0.CNI
	values["image_v1_1"] = common.Config.OLM.Images.V1_1.CNI
	values["image_v2_0"] = common.Config.OLM.Images.V2_0.CNI
	values["imagePullSecrets"] = config.ImagePullSecrets
	// TODO: imagePullPolicy, resources

	var releases []string
	for _, version := range supportedVersions {
		releases = append(releases, version.String())
	}
	values["supportedReleases"] = releases

	// always install the latest version of the CNI image
	renderings, _, err = helm.RenderChart(path.Join(versions.DefaultVersion.GetChartsDir(), "istio_cni"), operatorNamespace, values)
	return
}

func internalProcessManifests(ctx context.Context, cl client.Client, rendering []manifest.Manifest) error {
	operatorNamespace := common.GetOperatorNamespace()

	controllerResources := common.ControllerResources{
		Client:            cl,
		OperatorNamespace: operatorNamespace,
	}

	mp := helm.NewManifestProcessor(controllerResources, helm.NewPatchFactory(cl), "istio_cni", "TODO", "maistra-istio-operator", preProcessObject, postProcessObject)
	if err := mp.ProcessManifests(ctx, rendering, "istio_cni"); err != nil {
		return err
	}

	return nil
}

func preProcessObject(ctx context.Context, obj *unstructured.Unstructured) error {
	return nil
}

func postProcessObject(ctx context.Context, obj *unstructured.Unstructured) error {
	return nil
}
