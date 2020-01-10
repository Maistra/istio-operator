package common

import (
	"context"
	"strings"

	"github.com/ghodss/yaml"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"

	"k8s.io/helm/pkg/manifest"
	"k8s.io/helm/pkg/releaseutil"

	"k8s.io/kubernetes/pkg/kubectl"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ManifestProcessor struct {
	ResourceManager
	preprocessObject func(obj runtime.Object) error
	processNewObject func(obj runtime.Object) error

	appInstance, appVersion, owner string
}

func NewManifestProcessor(resourceManager ResourceManager, appInstance, appVersion, owner string, preprocessObjectFunc, postProcessObjectFunc func(obj runtime.Object) error) *ManifestProcessor {
	return &ManifestProcessor{
		ResourceManager:  resourceManager,
		preprocessObject: preprocessObjectFunc,
		processNewObject: postProcessObjectFunc,
		appInstance:      appInstance,
		appVersion:       appVersion,
		owner:            owner,
	}
}

func (p *ManifestProcessor) ProcessManifests(manifests []manifest.Manifest, component string) error {
	allErrors := []error{}

	origLogger := p.Log
	defer func() { p.Log = origLogger }()
	for _, man := range manifests {
		p.Log = origLogger.WithValues("manifest", man.Name)
		if !strings.HasSuffix(man.Name, ".yaml") {
			p.Log.V(2).Info("Skipping rendering of manifest")
			continue
		}
		p.Log.V(2).Info("Processing resources from manifest")
		// split the manifest into individual objects
		objects := releaseutil.SplitManifests(man.Content)
		for _, raw := range objects {
			rawJSON, err := yaml.YAMLToJSON([]byte(raw))
			if err != nil {
				p.Log.Error(err, "unable to convert raw data to JSON")
				allErrors = append(allErrors, err)
				continue
			}
			obj, err := runtime.Decode(p.JSONSerializer, rawJSON)
			if err != nil {
				p.Log.Error(err, "unable to decode object into Unstructured")
				allErrors = append(allErrors, err)
				continue
			}
			err = p.processObject(obj, component)
			if err != nil {
				allErrors = append(allErrors, err)
			}
		}
	}

	return utilerrors.NewAggregate(allErrors)
}

func (p *ManifestProcessor) processObject(obj runtime.Object, component string) error {
	origLogger := p.Log
	defer func() { p.Log = origLogger }()

	objMeta, err := meta.Accessor(obj)
	if err != nil {
		p.Log.Error(err, "could not access object metadata")
		return err
	}
	objType, err := meta.TypeAccessor(obj)
	if err != nil {
		p.Log.Error(err, "could not access object metadata")
		return err
	}
	key := v1.NewResourceKey(objMeta, objType)
	p.Log = origLogger.WithValues("Resource", key)

	_, err = meta.ListAccessor(obj)
	if err == nil {
		// it's a list
		items, err := meta.ExtractList(obj)
		if err != nil {
			p.Log.Error(err, "error extracting List items")
			return err
		}
		allErrors := []error{}
		for _, item := range items {
			err = p.processObject(item, component)
			if err != nil {
				allErrors = append(allErrors, err)
			}
		}
		return utilerrors.NewAggregate(allErrors)
	}

	p.addMetadata(objMeta, component)

	p.Log.V(2).Info("beginning reconciliation of resource", "ResourceKey", key)

	err = p.preprocessObject(obj)
	if err != nil {
		p.Log.Error(err, "error preprocessing object")
		return err
	}

	err = kubectl.CreateApplyAnnotation(obj, p.JSONSerializer)
	if err != nil {
		p.Log.Error(err, "error adding apply annotation to object")
	}

	receiver, err := key.ToRuntimeObject(p.Scheme)
	if err != nil {
		p.Log.Error(err, "could not create receiver for resource")
		// this should never happen
		return err
	}
	objectKey, err := client.ObjectKeyFromObject(receiver)
	if err != nil {
		p.Log.Error(err, "client.ObjectKeyFromObject() failed for resource")
		// This can only happen if reciever isn't a runtime.Object
		// i.e. this should never happen
		return err
	}

	var patch Patch

	err = p.Client.Get(context.TODO(), objectKey, receiver)
	if err != nil {
		if errors.IsNotFound(err) {
			p.Log.Info("creating resource")
			err = p.Client.Create(context.TODO(), obj)
			if err == nil {
				// special handling
				if err := p.processNewObject(obj); err != nil {
					// just log for now
					p.Log.Error(err, "error during postprocessing of new resource")
				}
			} else {
				p.Log.Error(err, "error during creation of new resource")
			}
		}
	} else if patch, err = p.PatchFactory.CreatePatch(receiver, obj); err == nil && patch != nil {
		p.Log.Info("updating existing resource")
		_, err = patch.Apply()
		if errors.IsInvalid(err) {
			// patch was invalid, try delete/create
			p.Log.Info("patch failed.  attempting to delete and recreate the resource")
			if deleteErr := p.Client.Delete(context.TODO(), obj, client.PropagationPolicy(metav1.DeletePropagationBackground)); deleteErr == nil {
				// we need to remove the resource version, which was updated by the patching process
				objMeta.SetResourceVersion("")
				if createErr := p.Client.Create(context.TODO(), obj); createErr == nil {
					p.Log.Info("successfully recreated resource after patch failure")
					err = nil
				} else {
					p.Log.Error(createErr, "error trying to recreate resource after patch failure")
				}
			} else {
				p.Log.Error(deleteErr, "error deleting resource for recreation")
			}
		}
	}
	p.Log.V(2).Info("resource reconciliation complete")
	if err != nil {
		p.Log.Error(err, "error occurred reconciling resource")
	}
	return err
}

func (p *ManifestProcessor) addMetadata(obj metav1.Object, component string) {
	labels := map[string]string{
		// add app labels
		KubernetesAppNameKey:      component,
		KubernetesAppInstanceKey:  p.appInstance,
		KubernetesAppVersionKey:   p.appVersion,
		KubernetesAppComponentKey: component,
		KubernetesAppPartOfKey:    "istio",
		KubernetesAppManagedByKey: "maistra-istio-operator",
		// legacy
		// add owner label
		OwnerKey: p.owner,
	}
	SetLabels(obj, labels)
}
