package datastore_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// In this test we make sure that datastore can successfully process
// registrymetadata passed to it.
func TestWrite(t *testing.T) {
	packagesWant := []string{
		"amq-streams",
		"etcd",
		"federationv2",
		"prometheus",
		"service-catalog",
	}

	opsrc := &v1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID("123456"),
		},
	}

	metadata := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Repository: "amq-streams",
		},
		&datastore.RegistryMetadata{
			Repository: "etcd",
		},
		&datastore.RegistryMetadata{
			Repository: "federationv2",
		},
		&datastore.RegistryMetadata{
			Repository: "prometheus",
		},
		&datastore.RegistryMetadata{
			Repository: "service-catalog",
		},
	}

	ds := datastore.New()
	count, err := ds.Write(opsrc, metadata)
	require.NoError(t, err)
	require.Equal(t, len(packagesWant), count)

	list := ds.GetPackageIDs()
	packagesGot := strings.Split(list, ",")
	assert.ElementsMatch(t, packagesWant, packagesGot)
}

// In this test we make sure that, if we write an opsrc to the datastore,
// when we do a read the metadata that associates the repository to the opsrc
// is maintained.
func TestReadOpsrcMeta(t *testing.T) {
	opsrc := &v1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("123456"),
			Name:      "operators-opsrc",
			Namespace: "operators",
		},
		Spec: v1.OperatorSourceSpec{
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "registry-namespace",
		},
	}

	metadata := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Repository: "amq-streams",
			Namespace:  "operators",
		},
		&datastore.RegistryMetadata{
			Repository: "etcd",
			Namespace:  "operators",
		},
	}

	ds := datastore.New()
	_, err := ds.Write(opsrc, metadata)
	require.NoError(t, err)

	opsrcmeta, err := ds.Read(opsrc.Name, "amq-streams")
	require.NoError(t, err)
	assert.Equal(t, "https://quay.io/cnr", opsrcmeta.Endpoint)
	assert.Equal(t, "registry-namespace", opsrcmeta.RegistryNamespace)

	opsrcmeta, err = ds.Read(opsrc.Name, "etcd")
	require.NoError(t, err)
	assert.Equal(t, "https://quay.io/cnr", opsrcmeta.Endpoint)
	assert.Equal(t, "registry-namespace", opsrcmeta.RegistryNamespace)
}

// In this test we make sure that we properly relate multiple opsrcs
// to the correct repositories.
func TestReadOpsrcMetaMultipleOpsrc(t *testing.T) {
	opsrc := &v1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("123456"),
			Name:      "operators-opsrc",
			Namespace: "operators",
		},
		Spec: v1.OperatorSourceSpec{
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "registry-namespace",
		},
	}

	metadata := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Repository: "amq-streams",
			Namespace:  "operators",
		},
		&datastore.RegistryMetadata{
			Repository: "etcd",
			Namespace:  "operators",
		},
	}

	ds := datastore.New()
	_, err := ds.Write(opsrc, metadata)
	require.NoError(t, err)

	opsrc2 := &v1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("456789"),
			Name:      "operators-different",
			Namespace: "operators",
		},
		Spec: v1.OperatorSourceSpec{
			Endpoint:          "https://quay-diff.io/cnr",
			RegistryNamespace: "registry-namespace-diff",
		},
	}

	metadata = []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Repository: "federationv2",
			Namespace:  "operators",
		},
	}

	_, err = ds.Write(opsrc2, metadata)
	require.NoError(t, err)

	opsrcmeta, err := ds.Read(opsrc.Name, "amq-streams")
	require.NoError(t, err)
	assert.Equal(t, "https://quay.io/cnr", opsrcmeta.Endpoint)
	assert.Equal(t, "registry-namespace", opsrcmeta.RegistryNamespace)

	opsrcmeta, err = ds.Read(opsrc.Name, "etcd")
	require.NoError(t, err)
	assert.Equal(t, "https://quay.io/cnr", opsrcmeta.Endpoint)
	assert.Equal(t, "registry-namespace", opsrcmeta.RegistryNamespace)

	opsrcmeta, err = ds.Read(opsrc2.Name, "federationv2")
	require.NoError(t, err)
	assert.Equal(t, "https://quay-diff.io/cnr", opsrcmeta.Endpoint)
	assert.Equal(t, "registry-namespace-diff", opsrcmeta.RegistryNamespace)

}

// In this test we make sure that, if we have a datastore, we can search for
// an opsrc that contains a list of packages.
func TestSearchForSource(t *testing.T) {
	opsrc := &v1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("123456"),
			Name:      "operators-opsrc",
			Namespace: "operators",
		},
		Spec: v1.OperatorSourceSpec{
			Endpoint:          "https://quay.io/cnr",
			RegistryNamespace: "registry-namespace",
		},
	}

	metadata := []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Repository: "amq-streams",
			Namespace:  "operators",
		},
		&datastore.RegistryMetadata{
			Repository: "etcd",
			Namespace:  "operators",
		},
	}

	ds := datastore.New()
	_, err := ds.Write(opsrc, metadata)
	require.NoError(t, err)

	opsrc2 := &v1.OperatorSource{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("456789"),
			Name:      "operators-different",
			Namespace: "operators",
		},
		Spec: v1.OperatorSourceSpec{
			Endpoint:          "https://quay-diff.io/cnr",
			RegistryNamespace: "registry-namespace-diff",
		},
	}

	metadata = []*datastore.RegistryMetadata{
		&datastore.RegistryMetadata{
			Repository: "federationv2",
			Namespace:  "operators",
		},
		&datastore.RegistryMetadata{
			Repository: "etcd",
			Namespace:  "operators",
		},
	}

	_, err = ds.Write(opsrc2, metadata)
	require.NoError(t, err)

	// Search for a source that contains the amq-streams package.
	source, err := ds.SearchForSource([]string{"amq-streams"})
	require.NoError(t, err)
	assert.Equal(t, opsrc.Name, source)

	// Search for a source that contains both the amq-streams and etcd package.
	source, err = ds.SearchForSource([]string{"amq-streams", "etcd"})
	require.NoError(t, err)
	assert.Equal(t, opsrc.Name, source)

	// Search for a source that contains the amq-streams package.
	source, err = ds.SearchForSource([]string{"federationv2"})
	require.NoError(t, err)
	assert.Equal(t, opsrc2.Name, source)

	// Search for a package contained in multiple sources.
	source, err = ds.SearchForSource([]string{"etcd"})
	assert.NoError(t, err)

	// Search for a list of packages that aren't contained in a single source.
	packages := []string{"amq-streams", "etcd", "federationv2"}
	source, err = ds.SearchForSource(packages)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("Unable to resolve the source - no source contains the requested package(s) %v", packages), err.Error())

	// Search for a package that isn't contained in any existing source.
	packages = []string{"missing-package"}
	source, err = ds.SearchForSource(packages)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("Unable to resolve the source - no source contains the requested package(s) %v", packages), err.Error())
}
