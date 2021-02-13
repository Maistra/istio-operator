package versions

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/manifest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/common"
	"github.com/maistra/istio-operator/pkg/controller/common/cni"
	ver "github.com/maistra/istio-operator/pkg/version"
)

const (
	// InvalidVersion is not a valid version
	InvalidVersion version = iota
	// V1_0 -> v1.0
	V1_0
	// V1_1 -> v1.1
	V1_1
	// V2_0 -> v2.0
	V2_0
	// V2_1 -> v2.1
	V2_1
	// Add new versions here, above lastKnownVersion. Remember to add a string mapping in init() below
	lastKnownVersion version = iota - 1
)

func init() {
	versionToString = map[version]string{
		InvalidVersion: "InvalidVersion",
		V1_0:           "v1.0",
		V1_1:           "v1.1",
		V2_0:           "v2.0",
		V2_1:           "v2.1",
	}

	versionToStrategy = map[version]VersionStrategy{
		InvalidVersion: &invalidVersionStrategy{InvalidVersion},
		V1_0:           &versionStrategyV1_0{version: V1_0},
		V1_1:           &versionStrategyV1_1{version: V1_1},
		V2_0:           &versionStrategyV2_0{V2_0},
		V2_1:           &versionStrategyV2_1{V2_1},
	}

	versionToCNINetwork = map[version]string{
		InvalidVersion: "",
		V1_0:           "istio-cni",
		V1_1:           "v1-1-istio-cni",
		V2_0:           "v2-0-istio-cni",
		V2_1:           "v2-1-istio-cni",
	}

	for v, str := range versionToString {
		if v != InvalidVersion {
			stringToVersion[str] = v
			if v == V1_0 {
				// special handling for legacy case
				stringToVersion[""] = v
			}
		}
	}
	minimumSupportedVersion := ver.Info.MinimumSupportedVersion
	minVersion := stringToVersion[minimumSupportedVersion]
	if minVersion == InvalidVersion {
		panic(fmt.Sprintf("invalid minimum supported version: %v", minimumSupportedVersion))
	}

	for v := range versionToString {
		if v >= minVersion {
			supportedVersions = append(supportedVersions, v)
			supportedVersionNames = append(supportedVersionNames, v.String())
		}
	}
}

const (
	// DefaultVersion to use for new resources which have no version specified.
	DefaultVersion = lastKnownVersion
	// LegacyVersion to use with existing resources which have no version specified.
	LegacyVersion = V1_0
)

// Version represents a version of a control plane, major.minor, usually
// identified as something like v1.0.  Version objects are guaranteed to be
// sequentually ordered from oldest to newest.
type Version interface {
	fmt.Stringer
	// Version returns the internal version representation
	Version() version
	// Compare compares this version with another version.  If other is an older
	// version, a positive value will be returned.  If other is a newer version,
	// a negative value is returned.  If other is the same version, zero is
	// returned.
	Compare(other Version) int
	// Strategy provides a customizations specific to this version.
	Strategy() VersionStrategy
	GetChartsDir() string
	GetUserTemplatesDir() string
	GetDefaultTemplatesDir() string
	GetCNINetworkName() string
	IsSupported() bool
}

// ValidationStrategy is an interface used by the validating webhook for validating SMCP resources.
type ValidationStrategy interface {
	ValidateV1(ctx context.Context, cl client.Client, smcp *v1.ServiceMeshControlPlane) error
	ValidateV2(ctx context.Context, cl client.Client, meta *metav1.ObjectMeta, spec *v2.ControlPlaneSpec) error
	ValidateDowngrade(ctx context.Context, cl client.Client, smcp metav1.Object) error
	ValidateUpgrade(ctx context.Context, cl client.Client, smcp metav1.Object) error
}

// RenderingStrategy is an interface used by the reconciler to manage rendering of charts.
type RenderingStrategy interface {
	GetChartInstallOrder() [][]string
	SetImageValues(ctx context.Context, cr *common.ControllerResources, smcp *v1.ControlPlaneSpec) error
	Render(ctx context.Context, cr *common.ControllerResources, cniConfig cni.Config, smcp *v2.ServiceMeshControlPlane) (map[string][]manifest.Manifest, error)
	// for testing purposes
	ApplyProfiles(ctx context.Context, cr *common.ControllerResources, smcpSpec *v1.ControlPlaneSpec, targetNamespace string) (v1.ControlPlaneSpec, error)
}

// VersionStrategy provides encapsulates customization required for a particular
// version.
type VersionStrategy interface {
	Version
	RenderingStrategy
	ValidationStrategy
}

// GetSupportedVersions returns a list of versions supported by this operator
func GetSupportedVersions() []Version {
	return supportedVersions
}

func GetSupportedVersionNames() []string {
	return supportedVersionNames
}

type version int

var _ Version = version(0)

func (v version) String() string {
	if str, ok := versionToString[v]; ok {
		return str
	}
	panic(fmt.Sprintf("invalid version: %d", v))
}

func (v version) Compare(other Version) int {
	return int(v.Version() - other.Version())
}

func (v version) Version() version {
	return v
}

func (v version) Strategy() VersionStrategy {
	if strategy, ok := versionToStrategy[v]; ok {
		return strategy
	}
	panic(fmt.Sprintf("invalid version: %d", v))
}

func (v version) GetCNINetworkName() string {
	if network, ok := versionToCNINetwork[v]; ok {
		return network
	}
	panic(fmt.Sprintf("invalid version: %d", v))
}

func (v version) IsSupported() (supported bool) {
	for _, version := range supportedVersions {
		if version == v {
			supported = true
			return
		}
	}
	return
}

// ParseVersion returns a version for the specified string
func ParseVersion(str string) (version, error) {
	if v, ok := stringToVersion[str]; ok {
		return v, nil
	}
	return InvalidVersion, fmt.Errorf("invalid version: %s", str)
}

type nilVersionStrategy struct {
	version
}

var _ VersionStrategy = (*nilVersionStrategy)(nil)

func (v *nilVersionStrategy) SetImageValues(ctx context.Context, cr *common.ControllerResources, smcp *v1.ControlPlaneSpec) error {
	return nil
}
func (v *nilVersionStrategy) ValidateV1(ctx context.Context, cl client.Client, smcp *v1.ServiceMeshControlPlane) error {
	return nil
}
func (v *nilVersionStrategy) ValidateV2(ctx context.Context, cl client.Client, meta *metav1.ObjectMeta, spec *v2.ControlPlaneSpec) error {
	return nil
}
func (v *nilVersionStrategy) ValidateDowngrade(ctx context.Context, cl client.Client, smcp metav1.Object) error {
	return nil
}
func (v *nilVersionStrategy) ValidateUpgrade(ctx context.Context, cl client.Client, smcp metav1.Object) error {
	return nil
}
func (v *nilVersionStrategy) GetChartInstallOrder() [][]string {
	return nil
}
func (v *nilVersionStrategy) Render(ctx context.Context, cr *common.ControllerResources, cniConfig cni.Config, smcp *v2.ServiceMeshControlPlane) (map[string][]manifest.Manifest, error) {
	return nil, fmt.Errorf("nil version does not support rendering")
}

type invalidVersionStrategy struct {
	version
}

var _ VersionStrategy = (*invalidVersionStrategy)(nil)

func (v *invalidVersionStrategy) SetImageValues(ctx context.Context, cr *common.ControllerResources, smcp *v1.ControlPlaneSpec) error {
	return fmt.Errorf("invalid version: %s", v.version)
}
func (v *invalidVersionStrategy) ValidateV1(ctx context.Context, cl client.Client, smcp *v1.ServiceMeshControlPlane) error {
	return fmt.Errorf("invalid version: %s", v.version)
}
func (v *invalidVersionStrategy) ValidateV2(ctx context.Context, cl client.Client, meta *metav1.ObjectMeta, spec *v2.ControlPlaneSpec) error {
	return fmt.Errorf("invalid version: %s", v.version)
}
func (v *invalidVersionStrategy) ValidateDowngrade(ctx context.Context, cl client.Client, smcp metav1.Object) error {
	return fmt.Errorf("invalid version: %s", v.version)
}
func (v *invalidVersionStrategy) ValidateUpgrade(ctx context.Context, cl client.Client, smcp metav1.Object) error {
	return fmt.Errorf("invalid version: %s", v.version)
}
func (v *invalidVersionStrategy) GetChartInstallOrder() [][]string {
	return nil
}

func (v *invalidVersionStrategy) Render(ctx context.Context, cr *common.ControllerResources, cniConfig cni.Config, smcp *v2.ServiceMeshControlPlane) (map[string][]manifest.Manifest, error) {
	return nil, fmt.Errorf("invalid version: %s", v.version)
}

var versionToString = make(map[version]string)
var versionToCNINetwork = make(map[version]string)
var versionToStrategy = make(map[version]VersionStrategy)
var stringToVersion = make(map[string]version)
var supportedVersions []Version
var supportedVersionNames []string
