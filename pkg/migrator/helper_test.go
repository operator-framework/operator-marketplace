package migrator_test

import (
	"fmt"

	v1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestInstalledCscName = "installed-community-openshift-marketplace"

	TestDatastoreCatalogSourceName = "community-operators"

	TestOpsrcName = "test-operators"

	TestNameSpace = "openshift-marketplace"

	TestOpsrcPackages = "foo,bar"
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
