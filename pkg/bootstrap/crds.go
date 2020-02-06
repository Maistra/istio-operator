package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"

	"github.com/maistra/istio-operator/pkg/controller/common"
	"github.com/maistra/istio-operator/pkg/controller/hacks"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/helm/pkg/releaseutil"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var crdMutex sync.Mutex // ensure two workers don't deploy CRDs at same time

// InstallCRDs makes sure all CRDs have been installed.  CRDs are located from
// files in controller.HelmDir/istio-init/files
func InstallCRDs(ctx context.Context, cl client.Client, version string) error {
	// we only try to install them once.  if there's an error, we should probably
	// panic, as there's no way to recover.  for now, we just pass the error along.
	crdMutex.Lock()
	defer crdMutex.Unlock()

	log := common.LogFromContext(ctx)
	log.Info(fmt.Sprintf("ensuring %s CRDs are installed", version))
	crdPath := path.Join(common.Options.GetChartsDir(version), "istio-init/files")
	crdDir, err := os.Stat(crdPath)
	if err != nil || !crdDir.IsDir() {
		return fmt.Errorf("Cannot locate any CRD files in %s", crdPath)
	}
	err = filepath.Walk(crdPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		return processCRDFile(ctx, cl, path)
	})
	if err != nil {
		return err
	}
	return installCRDRole(ctx, cl)
}

func installCRDRole(ctx context.Context, cl client.Client) error {
	crdRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-admin",
			Labels: map[string]string{
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"config.istio.io",
					"networking.istio.io",
					"authentication.istio.io",
					"rbac.istio.io",
					"authentication.maistra.io",
					"rbac.maistra.io",
				},
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     []string{rbacv1.VerbAll},
			},
		},
	}
	if err := cl.Get(ctx, client.ObjectKey{Name: crdRole.Name}, crdRole); err == nil {
		return nil
	} else if errors.IsNotFound(err) {
		return cl.Create(ctx, crdRole)
	} else {
		return err
	}
}

func processCRDFile(ctx context.Context, cl client.Client, fileName string) error {
	log := common.LogFromContext(ctx)
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(file)
	if err != nil {
		return err
	}

	allErrors := []error{}
	for index, raw := range releaseutil.SplitManifests(string(buf.Bytes())) {
		crd, err := decodeCRD(common.NewContextWithLog(ctx, log.WithValues("file", fileName, "index", index)), raw)
		if err != nil {
			allErrors = append(allErrors, err)
		} else if crd != nil { // crd is nil when the object in the file isn't a CRD
			err = createCRD(common.NewContextWithLog(ctx, log.WithValues("CRD", crd.GetName())), cl, crd)
			if err != nil {
				allErrors = append(allErrors, err)
			}
		}
	}
	return utilerrors.NewAggregate(allErrors)
}

func decodeCRD(ctx context.Context, raw string) (*unstructured.Unstructured, error) {
	log := common.LogFromContext(ctx)
	rawJSON, err := yaml.YAMLToJSON([]byte(raw))
	if err != nil {
		log.Error(err, "unable to convert raw data to JSON")
		return nil, err
	}
	obj := &unstructured.Unstructured{}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(rawJSON, nil, obj)
	if err != nil {
		log.Error(err, "unable to decode object into Unstructured")
		return nil, err
	}
	if obj.GroupVersionKind().GroupKind().String() == "CustomResourceDefinition.apiextensions.k8s.io" {
		return obj, nil
	} else {
		return nil, nil
	}
}

func createCRD(ctx context.Context, cl client.Client, crd *unstructured.Unstructured) error {
	log := common.LogFromContext(ctx)
	existingCrd := &unstructured.Unstructured{}
	existingCrd.SetGroupVersionKind(crd.GroupVersionKind())
	existingCrd.SetName(crd.GetName())
	err := cl.Get(ctx, client.ObjectKey{Name: crd.GetName()}, existingCrd) // TODO: replace Unstructured with actual type
	if err == nil {
		newVersion, err := getMaistraVersion(crd)
		if err != nil {
			return fmt.Errorf("Could not determine version of new CRD %s: %v", crd.GetName(), err)
		}
		existingVersion, err := getMaistraVersion(existingCrd)
		if err != nil {
			log.Info("Could not determine version of existing CRD", "error", err)
			existingVersion = nil
		}
		if existingVersion == nil || existingVersion.LessThan(newVersion) {
			log.Info("CRD exists, but is old or has no version label. Replacing with newer version.")

			patchedCrd, err := getPatchedCrd(existingCrd, crd)
			if err != nil {
				return err
			}
			if patchedCrd != nil { // patchedCrd is nil when the existing and new CRDs are identical
				err = cl.Update(ctx, patchedCrd)
				if hacks.IsTypeObjectProblemInCRDSchemas(err) {
					err = hacks.RemoveTypeObjectFieldsFromCRDSchema(ctx, patchedCrd)
					if err != nil {
						return err
					}
					err = cl.Update(ctx, patchedCrd)
				}
				if err != nil {
					log.Error(err, "error updating CRD")
					return err
				}
			}

		} else {
			log.Info("CRD exists")
		}
		return nil
	}
	if errors.IsNotFound(err) {
		log.Info("creating CRD")
		err = cl.Create(ctx, crd)
		if hacks.IsTypeObjectProblemInCRDSchemas(err) {
			err = hacks.RemoveTypeObjectFieldsFromCRDSchema(ctx, crd)
			if err != nil {
				return err
			}
			err = cl.Create(ctx, crd)
		}
		if err != nil {
			log.Error(err, "error creating CRD")
			return err
		}
		return nil
	}
	return err
}

func getPatchedCrd(existingCrd *unstructured.Unstructured, crd *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	patchedCrd, err := common.GetPatchedObject(existingCrd, crd)
	if err != nil || patchedCrd == nil {
		return nil, err
	}
	if newUnstructured, ok := patchedCrd.(*unstructured.Unstructured); ok {
		return newUnstructured, nil
	}
	return nil, fmt.Errorf("could not decode unstructured object:\n%v", patchedCrd)
}

func getMaistraVersion(crd *unstructured.Unstructured) (*semver.Version, error) {
	return semver.NewVersion(crd.GetLabels()["maistra-version"])
}
