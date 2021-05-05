package testgroups

import (
	"testing"

	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
)

// NoSetupTestGroup runs test suites that do not require any resources upfront
func NoSetupTestGroup(t *testing.T) {
	if isConfigAPIPresent, _ := helpers.EnsureConfigAPIIsAvailable(); isConfigAPIPresent {
		t.Run("operatorhub-test-suite", testsuites.OperatorHubTests)
		t.Run("default-catalogsource-test-suite", testsuites.DefaultCatsrc)
	}
}
