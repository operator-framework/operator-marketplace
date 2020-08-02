package testgroups

import (
	"testing"

	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-marketplace/test/testsuites"
)

// NoSetupTestGroup runs test suites that do not require any resources upfront
func NoSetupTestGroup(t *testing.T) {

	// Run the test suites.
	t.Run("invalid-operator-source-test-suite", testsuites.InvalidOpSrc)
	t.Run("delete-operator-source-test-suite", testsuites.DeleteOpSrc)
	if isConfigAPIPresent, _ := helpers.EnsureConfigAPIIsAvailable(); isConfigAPIPresent == true {
		t.Run("operatorhub-test-suite", testsuites.OperatorHubTests)
	}
}
