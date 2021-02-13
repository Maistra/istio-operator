package validation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	maistrav2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/common"
	webhookcommon "github.com/maistra/istio-operator/pkg/controller/servicemesh/webhooks/common"
	"github.com/maistra/istio-operator/pkg/controller/versions"
)

type ControlPlaneValidator struct {
	client          client.Client
	decoder         *admission.Decoder
	namespaceFilter webhookcommon.NamespaceFilter
}

func NewControlPlaneValidator(namespaceFilter webhookcommon.NamespaceFilter) *ControlPlaneValidator {
	return &ControlPlaneValidator{
		namespaceFilter: namespaceFilter,
	}
}

var _ admission.Handler = (*ControlPlaneValidator)(nil)
var _ inject.Client = (*ControlPlaneValidator)(nil)
var _ admission.DecoderInjector = (*ControlPlaneValidator)(nil)

func (v *ControlPlaneValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := logf.Log.WithName("smcp-validator").
		WithValues("ServiceMeshControlPlane", webhookcommon.ToNamespacedName(&req.AdmissionRequest))

	// do we care about this object?
	if !v.namespaceFilter.Watching(req.Namespace) {
		logger.Info(fmt.Sprintf("operator is not watching namespace '%s'", req.Namespace))
		return admission.Allowed("")
	}

	if req.Namespace == common.GetOperatorNamespace() {
		return validationFailedResponse(http.StatusBadRequest, metav1.StatusReasonBadRequest, fmt.Sprintf("service mesh may not be installed in the same project/namespace as the operator"))
	}

	smcpList := &maistrav2.ServiceMeshControlPlaneList{}
	err := v.client.List(ctx, smcpList, client.InNamespace(req.Namespace))
	if err != nil {
		logger.Error(err, "error listing smcp resources")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// verify single instance per namespace
	if (len(smcpList.Items) == 1 && smcpList.Items[0].Name != req.Name) || len(smcpList.Items) > 1 {
		return validationFailedResponse(http.StatusBadRequest, metav1.StatusReasonBadRequest, "only one service mesh may be installed per project/namespace")
	}

	smcprequest, err := v.decodeRequest(req, logger)
	if err != nil {
		logger.Error(err, "error decoding admission request")
		return admission.Errored(http.StatusBadRequest, err)
	} else if smcprequest.New().GetDeletionTimestamp() != nil {
		logger.Info("skipping deleted smcp resource")
		return admission.Allowed("")
	} else if ! smcprequest.NewVersion().IsSupported() {
		return validationFailedResponse(http.StatusBadRequest, metav1.StatusReasonBadRequest, fmt.Sprintf("Only '%v' versions are supported", versions.GetSupportedVersionNames()))
	} else if err := v.validateVersion(ctx, smcprequest.New(), smcprequest.NewVersion()); err != nil {
		return validationFailedResponse(http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
	}

	if req.AdmissionRequest.Operation == admissionv1beta1.Update {
		// verify update
		return v.validateUpdate(ctx, smcprequest.OldVersion(), smcprequest.NewVersion(), smcprequest.New(), logger)
	}

	return admission.ValidationResponse(true, "")
}

func (v *ControlPlaneValidator) decodeRequest(req admission.Request, logger logr.Logger) (smcprequest, error) {
	switch req.Kind.Version {
	case maistrav1.SchemeGroupVersion.Version:
		smcp := &maistrav1.ServiceMeshControlPlane{}
		err := v.decoder.Decode(req, smcp)
		if err != nil {
			logger.Error(err, "error decoding admission request")
			return nil, err
		}
		newVersion, err := versions.ParseVersion(smcp.Spec.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid Version specified; supported versions are: %v", versions.GetSupportedVersions())
		}
		var oldsmcp *maistrav1.ServiceMeshControlPlane
		oldVersion := versions.InvalidVersion
		if req.Operation == admissionv1beta1.Update {
			oldsmcp = &maistrav1.ServiceMeshControlPlane{}
			err = v.decoder.DecodeRaw(req.AdmissionRequest.OldObject, oldsmcp)
			if err != nil {
				logger.Error(err, "error decoding admission request")
				return nil, err
			}
			oldVersion, err = versions.ParseVersion(oldsmcp.Spec.Version)
			if err != nil {
				return nil, fmt.Errorf("invalid Version specified; supported versions are: %v", versions.GetSupportedVersions())
			}
		}
		return &smcpv1request{new: smcp, old: oldsmcp, newVersion: newVersion, oldVersion: oldVersion}, nil
	case maistrav2.SchemeGroupVersion.Version:
		smcp := &maistrav2.ServiceMeshControlPlane{}
		err := v.decoder.Decode(req, smcp)
		if err != nil {
			logger.Error(err, "error decoding admission request")
			return nil, err
		}
		newVersion, err := versions.ParseVersion(smcp.Spec.Version)
		if err != nil {
			return nil, fmt.Errorf("invalid Version specified; supported versions are: %v", versions.GetSupportedVersions())
		}
		var oldsmcp *maistrav2.ServiceMeshControlPlane
		oldVersion := versions.InvalidVersion
		if req.Operation == admissionv1beta1.Update {
			oldsmcp = &maistrav2.ServiceMeshControlPlane{}
			err = v.decoder.DecodeRaw(req.AdmissionRequest.OldObject, oldsmcp)
			if err != nil {
				logger.Error(err, "error decoding admission request")
				return nil, err
			}
			oldVersion, err = versions.ParseVersion(oldsmcp.Spec.Version)
			if err != nil {
				return nil, fmt.Errorf("invalid Version specified; supported versions are: %v", versions.GetSupportedVersions())
			}
		}
		return &smcpv2request{new: smcp, old: oldsmcp, newVersion: newVersion, oldVersion: oldVersion}, nil
	default:
		return nil, fmt.Errorf("unkown resource type: %s", req.Kind.String())
	}
}

func (v *ControlPlaneValidator) validateVersion(ctx context.Context, obj metav1.Object, version versions.Version) error {
	switch smcp := obj.(type) {
	case *maistrav1.ServiceMeshControlPlane:
		switch version.Version() {
		// UndefinedVersion defaults to legacy v1.0
		case versions.V1_0:
			// no validation existed in 1.0, so we won't validate
			return nil
		}
		return version.Strategy().ValidateV1(ctx, v.client, smcp)
	case *maistrav2.ServiceMeshControlPlane:
		return version.Strategy().ValidateV2(ctx, v.client, &smcp.ObjectMeta, &smcp.Spec)
	default:
		return fmt.Errorf("unknown ServiceMeshControlPlane type: %T", smcp)
	}
}

func (v *ControlPlaneValidator) validateUpdate(ctx context.Context, oldVersion, newVersion versions.Version, new metav1.Object, logger logr.Logger) admission.Response {

	// The logic used here is that we only verify upgrade/downgrade between adjacent versions
	// If an upgrade/downgrade spans multiple versions, the validation for upgrade/downgrade
	// between adjacent versions is chained together, e.g. 1.0 -> 1.3, we'd verify
	// upgrade from 1.0 -> 1.1, then 1.1 -> 1.2, then 1.2 -> 1.3.  If all of those
	// were successful, validation succeeds.  This approach may breakdown if a feature
	// was removed and subsequently reintroduced (e.g. validation from 1.0 -> 1.1
	// fails because feature X is no longer supported, but was added back in 1.3).
	if oldVersion.Version() < newVersion.Version() {
		for version := oldVersion.Version() + 1; version <= newVersion.Version(); version++ {
			if err := version.Strategy().ValidateUpgrade(ctx, v.client, new); err != nil {
				return validationFailedResponse(http.StatusBadRequest, metav1.StatusReasonBadRequest, fmt.Sprintf("cannot upgrade control plane from version %s to %s: %s", oldVersion.String(), newVersion.String(), err))
			}
		}
	} else {
		for version := oldVersion.Version(); version > newVersion.Version(); version-- {
			if err := version.Strategy().ValidateDowngrade(ctx, v.client, new); err != nil {
				return validationFailedResponse(http.StatusBadRequest, metav1.StatusReasonBadRequest, fmt.Sprintf("cannot downgrade control plane from version %s to %s: %s", oldVersion.String(), newVersion.String(), err))
			}
		}
	}

	return admission.Allowed("")
}

// InjectClient injects the client.
func (v *ControlPlaneValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *ControlPlaneValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

type smcprequest interface {
	New() metav1.Object
	NewVersion() versions.Version
	Old() metav1.Object
	OldVersion() versions.Version
}

type smcpv1request struct {
	new        *maistrav1.ServiceMeshControlPlane
	old        *maistrav1.ServiceMeshControlPlane
	newVersion versions.Version
	oldVersion versions.Version
}

var _ smcprequest = (*smcpv1request)(nil)

func (smcp *smcpv1request) New() metav1.Object {
	if smcp == nil {
		return nil
	}
	return smcp.new
}

func (smcp *smcpv1request) NewVersion() versions.Version {
	if smcp == nil {
		return versions.InvalidVersion
	}
	return smcp.newVersion
}

func (smcp *smcpv1request) Old() metav1.Object {
	if smcp == nil {
		return nil
	}
	return smcp.old
}

func (smcp *smcpv1request) OldVersion() versions.Version {
	if smcp == nil {
		return versions.InvalidVersion
	}
	return smcp.oldVersion
}

type smcpv2request struct {
	new        *maistrav2.ServiceMeshControlPlane
	old        *maistrav2.ServiceMeshControlPlane
	newVersion versions.Version
	oldVersion versions.Version
}

var _ smcprequest = (*smcpv2request)(nil)

func (smcp *smcpv2request) New() metav1.Object {
	if smcp == nil {
		return nil
	}
	return smcp.new
}

func (smcp *smcpv2request) NewVersion() versions.Version {
	if smcp == nil {
		return versions.InvalidVersion
	}
	return smcp.newVersion
}

func (smcp *smcpv2request) Old() metav1.Object {
	if smcp == nil {
		return nil
	}
	return smcp.old
}

func (smcp *smcpv2request) OldVersion() versions.Version {
	if smcp == nil {
		return versions.InvalidVersion
	}
	return smcp.oldVersion
}
