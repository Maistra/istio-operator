package version

import (
	"fmt"
	"runtime"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
)

var (
	buildVersion     = "unknown"
	buildGitRevision = "unknown"
	buildStatus      = "unknown"
	buildTag         = "unknown"

	// Minimum supported mesh version (nil (all), "v1_0", "v1_1", "v2_0" etc)
	minimumSupportedVersion = ""

	// Info exports the build version information.
	Info BuildInfo
)

// BuildInfo describes version information about the binary build.
type BuildInfo struct {
	Version                 string
	GitRevision             string
	BuildStatus             string
	GitTag                  string
	GoVersion               string
	GoArch                  string
	OperatorSDK             string
	MinimumSupportedVersion string
}

func (b BuildInfo) String() string {
	return fmt.Sprintf("%#v", b)
}

func init() {
	Info = BuildInfo{
		Version:                 buildVersion,
		GitRevision:             buildGitRevision,
		BuildStatus:             buildStatus,
		GitTag:                  buildTag,
		GoVersion:               runtime.Version(),
		GoArch:                  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		OperatorSDK:             sdkVersion.Version,
		MinimumSupportedVersion: minimumSupportedVersion,
	}
}
