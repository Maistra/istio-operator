package common

import (
	"context"
	"fmt"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/kubectl"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PatchFactory wraps the objects needed to create Patch objects.
type PatchFactory struct {
	client     client.Client
	serializer runtime.Serializer
	encoder    runtime.Encoder
}

// Patch represents a "patch" for an object
type Patch interface {
	Apply() (runtime.Object, error)
}

// NewPatchFactory creates a new PatchFactory
func NewPatchFactory(k8sClient client.Client, scheme *runtime.Scheme) *PatchFactory {
	serializer := json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false)
	return &PatchFactory{client: k8sClient, serializer: serializer, encoder: unstructured.JSONFallbackEncoder{Encoder: serializer}}
}

// CreatePatch creates a patch based on the current and new versions of an object
func (p *PatchFactory) CreatePatch(current, new runtime.Object) (Patch, error) {
	patch := &basicPatch{client: p.client}
	currentAccessor, err := meta.Accessor(current)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("cannot create object accessor for current object:\n%v", current))
	} else if newAccessor, err := meta.Accessor(new); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("cannot create object accessor for new object:\n%v", new))
	} else {
		newAccessor.SetResourceVersion(currentAccessor.GetResourceVersion())
	}

	// Serialize the current configuration of the object.
	currentBytes, err := runtime.Encode(p.encoder, current)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not serialize current object into raw json:\n%v", current))
	}

	// Serialize the new configuration of the object.
	newBytes, err := runtime.Encode(p.encoder, new)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not serialize new object into raw json:\n%v", new))
	}

	// Retrieve the original configuration of the object.
	originalBytes, err := kubectl.GetOriginalConfiguration(current)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to retrieve original configuration from current object:\n%v", current))
	}

	var gvk schema.GroupVersionKind
	if currentTypeAccessor, err := meta.TypeAccessor(current); err == nil {
		gvk = schema.FromAPIVersionAndKind(currentTypeAccessor.GetAPIVersion(), currentTypeAccessor.GetKind())
	} else {
		return nil, errors.Wrap(err, fmt.Sprintf("error getting GroupVersionKind for object"))
	}

	// if we can get a versioned object from the scheme, we can use the strategic patching mechanism
	// (i.e. take advantage of patchStrategy in the type)
	versionedObject, err := scheme.Scheme.New(gvk)
	if err != nil {
		// json merge patch
		preconditions := []mergepatch.PreconditionFunc{
			mergepatch.RequireKeyUnchanged("apiVersion"),
			mergepatch.RequireKeyUnchanged("kind"),
			mergepatch.RequireMetadataKeyUnchanged("name"),
		}
		// prevent precondition errors for cluster scoped resources
		if currentAccessor.GetNamespace() != "" {
			preconditions = append(preconditions, mergepatch.RequireMetadataKeyUnchanged("namespace"))
		}
		patchBytes, err := jsonmergepatch.CreateThreeWayJSONMergePatch(originalBytes, newBytes, currentBytes, preconditions...)
		if err != nil {
			if mergepatch.IsPreconditionFailed(err) {
				return nil, errors.Wrap(err, fmt.Sprintf("cannot change apiVersion, kind, name, or namespace fields"))
			}
			return nil, errors.Wrap(err, fmt.Sprintf("could not create patch for object, original:\n%v\ncurrent:\n%v\nnew:\n%v", originalBytes, currentBytes, newBytes))
		}
		if string(patchBytes) == "{}" {
			// empty patch, nothing to do
			return nil, nil
		}
		newBytes, err := jsonpatch.MergePatch(currentBytes, patchBytes)
		if err != nil {
			return nil, err
		}
		newObj, err := runtime.Decode(p.serializer, newBytes)
		if err != nil {
			return nil, err
		}
		if reflect.DeepEqual(newObj, current) {
			return nil, nil
		}
		patch.newObj = newObj
	} else {
		// XXX: if we fail to create a strategic patch, should we fall back to json merge patch?
		// strategic merge patch
		lookupPatchMeta, err := strategicpatch.NewPatchMetaFromStruct(versionedObject)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve patch metadata for object: %s", gvk.String()))
		}
		patchBytes, err := strategicpatch.CreateThreeWayMergePatch(originalBytes, newBytes, currentBytes, lookupPatchMeta, true)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not create patch for object, original:\n%v\ncurrent:\n%v\nnew:\n%v", originalBytes, currentBytes, newBytes))
		}
		if string(patchBytes) == "{}" {
			// empty patch, nothing to do
			return nil, nil
		}
		newBytes, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(currentBytes, patchBytes, lookupPatchMeta)
		if err != nil {
			return nil, err
		}
		newObj, err := runtime.Decode(p.serializer, newBytes)
		if err != nil {
			return nil, err
		}
		if reflect.DeepEqual(newObj, current) {
			return nil, nil
		}
		patch.newObj = newObj
	}

	return patch, nil
}

type basicPatch struct {
	client client.Client
	newObj runtime.Object
}

func (p *basicPatch) Apply() (runtime.Object, error) {
	err := p.client.Update(context.TODO(), p.newObj)
	return p.newObj, err
}
