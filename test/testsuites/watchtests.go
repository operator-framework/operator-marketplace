package testsuites

import (
	"fmt"
	"testing"

	olm "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	cscErrMsg   = "CatalogSourceConfig child resource(s) were not recreated"
	opsrcErrMsg = "OperatorSource child resource(s) were not recreated"
)

// WatchTests is a test suite that ensure that the watches for child resources
// are firing correctly and the child resources are restored upon deletion.
func WatchTests(t *testing.T) {
	t.Run("restore-opsrc-catalogsource", testRestoreOpSrcCs)
	t.Run("restore-opsrc-deployment", testRestoreOpSrcDeployment)
	t.Run("restore-opsrc-service", testRestoreOpSrcService)
}

// testRestoreOpSrcCs tests that when a CatalogSource that is owned by an OperatorSource
// is restored upon deletion.
func testRestoreOpSrcCs(t *testing.T) {
	err := deleteCheckRestoreChild(t, olm.CatalogSourceKind, v1.OperatorSourceKind)
	assert.NoError(t, err, opsrcErrMsg)
}

// testRestoreOpSrcDeployment tests that when a Deployment that is owned by an OperatorSource
// is restored upon deletion.
func testRestoreOpSrcDeployment(t *testing.T) {
	err := deleteCheckRestoreChild(t, "Deployment", v1.OperatorSourceKind)
	assert.NoError(t, err, opsrcErrMsg)
}

// testRestoreOpSrcService tests that when a Service that is owned by an OperatorSourceKind
// is restored upon deletion.
func testRestoreOpSrcService(t *testing.T) {
	err := deleteCheckRestoreChild(t, "Service", v1.OperatorSourceKind)
	assert.NoError(t, err, opsrcErrMsg)
}

// deleteCheckRestoreChild constructs the child resource based on the object and
// deletes it. It then checks if the child resources were recreated.
func deleteCheckRestoreChild(t *testing.T, child string, owner string) error {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	require.NoError(t, err, "Could not get namespace")

	client := test.Global.Client

	var (
		obj             runtime.Object
		name            string
		targetNamespace string
	)
	switch owner {
	case v1.OperatorSourceKind:
		name = helpers.TestOperatorSourceName
		targetNamespace = namespace
	default:
		return fmt.Errorf("unknown owner %s", owner)
	}

	objMeta := meta.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}

	switch child {
	case olm.CatalogSourceKind:
		obj = &olm.CatalogSource{
			TypeMeta: meta.TypeMeta{
				Kind: olm.CatalogSourceKind,
			},
			ObjectMeta: meta.ObjectMeta{
				Name:      name,
				Namespace: targetNamespace,
			},
		}
	case "Deployment":
		obj = &apps.Deployment{
			TypeMeta: meta.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: objMeta,
		}
	case "Service":
		obj = &core.Service{
			TypeMeta: meta.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: objMeta,
		}
	default:
		return fmt.Errorf("unknown child %s", child)
	}

	// Delete the object
	err = helpers.DeleteRuntimeObject(client, obj)
	require.NoError(t, err, "Error deleting %s %s/%s", child, name, namespace)

	// Confirm child resources were recreated without errors which implies that the
	// owner resource was recreated
	return helpers.CheckChildResourcesCreated(client, name, namespace, targetNamespace, owner)
}
