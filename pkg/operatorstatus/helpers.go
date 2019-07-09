package operatorstatus

import (
	"os"
)

const (
	// releaseVersionEnvVar is an environment variable that is
	// expected to host the ClusterOperator version.
	releaseVersionEnvVar = "RELEASE_VERSION"
)

// getReleaseVersion returns the release version. If a release version is
// not found, then `OpenShift Independent Version` will be returned instead.
func getReleaseVersion() string {
	version := os.Getenv(releaseVersionEnvVar)
	if version == "" {
		return "OpenShift Independent Version"
	}
	return version
}
