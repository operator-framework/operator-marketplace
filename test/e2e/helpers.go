package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/test"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	retryInterval = time.Second * 5
	timeout       = time.Second * 60
)

// This function polls the cluster for a particular resource name and namespace
// If the request fails because of an IsNotFound error it retries until the specified timeout
// If it succeeds it sets the result runtime.Object to the requested object
func WaitForResult(t *testing.T, f *test.Framework, result runtime.Object, namespace, name string) error {
	namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(context.TODO(), namespacedName, result)
		if err != nil {
			if errors.IsNotFound(err) {
				t.Logf("Waiting for creation of %s runtime object\n", name)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	t.Logf("Runtime object %s has been created\n", name)
	return nil
}
