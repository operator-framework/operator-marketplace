package migrator_test

import (
	"fmt"
	"testing"

	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/migrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestInstalledCscName           = "installed-community-openshift-marketplace"
	TestDatastoreCatalogSourceName = "community-operators"
	TestOpsrcName                  = "test-operators"
	TestNameSpace                  = "openshift-marketplace"
	TestOpsrcPackages              = "foo,bar"
)

func helperNewOperatorSourceWithPackage(packages string) *v1.OperatorSource {
	return &v1.OperatorSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s",
				v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version),
			Kind: v1.OperatorSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestOpsrcName,
			Namespace: TestNameSpace,
		},
		Status: v1.OperatorSourceStatus{
			Packages: packages,
		},
	}
}

// TestMigrator_InfersDatastoreCatalogSource_Correctly tests if
// ExtractCsName extracts the CatalogSource name from a given
// CatalogSourceConfig name correctly
func TestMigrator_InfersDatastoreCatalogSource_Correctly(t *testing.T) {
	extractedCsName, err := migrator.ExtractCsName(TestInstalledCscName)
	require.NoError(t, err)
	assert.Equal(t, extractedCsName, TestDatastoreCatalogSourceName)
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
