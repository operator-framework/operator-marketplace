package operatorstatus

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReleaseVersion(t *testing.T) {
	assert.Equal(t, "OpenShift Independent Version", getReleaseVersion())
	expected := "v1"
	os.Setenv(releaseVersionEnvVar, expected)
	assert.Equal(t, expected, getReleaseVersion())
}
