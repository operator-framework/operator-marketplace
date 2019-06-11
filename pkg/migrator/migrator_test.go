package migrator_test

import (
	"github.com/operator-framework/operator-marketplace/pkg/migrator"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestMigrator_InfersDatastoreCatalogSource_Correctly tests if
// ExtractCsName extracts the CatalogSource name from a given
// CatalogSourceConfig name correctly
func TestMigrator_InfersDatastoreCatalogSource_Correctly(t *testing.T) {
	assert.Equal(t, migrator.ExtractCsName(TestInstalledCscName), TestDatastoreCatalogSourceName)
}

// TestMigrator_ReportsPackageInOpsrc_True tests if IsPackageInOpsrc
// reports True if a given package is present in a given OperatorSource.
func TestMigrator_ReportsPackageInOpsrc_True(t *testing.T) {
	assert.True(t, migrator.IsPackageInOpsrc("foo", helperNewOperatorSourceWithPackage(TestOpsrcPackages)))
}

// TestMigrator_ReportsPackageInOpsrc_False tests if IsPackageInOpsrc
// reports False if a given package is not present in a given OperatorSource.
func TestMigrator_ReportsPackageInOpsrc_False(t *testing.T) {
	assert.False(t, migrator.IsPackageInOpsrc("baz", helperNewOperatorSourceWithPackage(TestOpsrcPackages)))
}
