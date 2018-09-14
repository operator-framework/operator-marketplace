package appregistry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/stretchr/testify/require"
)

func TestRetrieveAll(t *testing.T) {
	factory := appregistry.NewClientFactory()

	client, err := factory.New("appregistry", "http://localhost:5000/cnr")
	require.NoError(t, err)

	packages, err := client.RetrieveAll()

	assert.NoError(t, err)
	assert.NotNil(t, packages)
}
