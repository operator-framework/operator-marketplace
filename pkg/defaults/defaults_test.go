package defaults

import (
	"testing"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetCatalogSourceImageTagOverride(t *testing.T) {
	tests := []struct {
		name          string
		versionString string
		wantTag       string
		wantErr       bool
	}{
		{
			name:          "empty version string",
			versionString: "",
			wantTag:       "",
			wantErr:       false,
		},
		{
			name:          "default snapshot version",
			versionString: "0.0.1-snapshot",
			wantTag:       "",
			wantErr:       false,
		},
		{
			name:          "valid OpenShift 4.23.0",
			versionString: "4.23.0",
			wantTag:       "v4.23",
			wantErr:       false,
		},
		{
			name:          "valid OpenShift 5.0.0",
			versionString: "5.0.0",
			wantTag:       "v5.0",
			wantErr:       false,
		},
		{
			name:          "version with pre-release",
			versionString: "4.21.0-rc.1",
			wantTag:       "v4.21",
			wantErr:       false,
		},
		{
			name:          "version with build metadata",
			versionString: "4.23.0+build123",
			wantTag:       "v4.23",
			wantErr:       false,
		},
		{
			name:          "zero major version (development)",
			versionString: "0.5.0",
			wantTag:       "",
			wantErr:       false,
		},
		{
			name:          "zero major and minor",
			versionString: "0.0.1",
			wantTag:       "",
			wantErr:       false,
		},
		{
			name:          "invalid semver - not a version",
			versionString: "not-a-version",
			wantTag:       "",
			wantErr:       true,
		},
		{
			name:          "invalid semver - v prefix",
			versionString: "v4.23.0",
			wantTag:       "",
			wantErr:       true,
		},
		{
			name:          "invalid semver - missing patch",
			versionString: "4.23",
			wantTag:       "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTag, err := GetCatalogSourceImageTagOverride(tt.versionString)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse version string")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantTag, gotTag)
		})
	}
}

func TestOverrideImageTag(t *testing.T) {
	tests := []struct {
		name             string
		catsrc           *olmv1alpha1.CatalogSource
		imageTagOverride string
		wantImage        string
		wantErr          bool
	}{
		{
			name:             "nil CatalogSource",
			catsrc:           nil,
			imageTagOverride: "v4.21",
			wantImage:        "",
			wantErr:          false,
		},
		{
			name: "empty imageTagOverride",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io/catalog:v4.20",
				},
			},
			imageTagOverride: "",
			wantImage:        "registry.io/catalog:v4.20",
			wantErr:          false,
		},
		{
			name: "empty image field",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       olmv1alpha1.CatalogSourceSpec{},
			},
			imageTagOverride: "v5.0",
			wantImage:        "",
			wantErr:          false,
		},
		{
			name: "tagged image override",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io/catalog:v4.23",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "registry.io/catalog:v5.0",
			wantErr:          false,
		},
		{
			name: "non-semver tagged image override",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io/catalog:latest",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "registry.io/catalog:v5.0",
			wantErr:          false,
		},
		{
			name: "untagged image override",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io/catalog",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "registry.io/catalog:v5.0",
			wantErr:          false,
		},
		{
			name: "digest-based image unchanged",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io/catalog@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "registry.io/catalog@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantErr:          false,
		},
		{
			name: "image with port",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io:5000/catalog:v4.23",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "registry.io:5000/catalog:v5.0",
			wantErr:          false,
		},
		{
			name: "image with nested path",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "registry.io/org/team/catalog:v4.23",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "registry.io/org/team/catalog:v5.0",
			wantErr:          false,
		},
		{
			name: "invalid image reference",
			catsrc: &olmv1alpha1.CatalogSource{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: olmv1alpha1.CatalogSourceSpec{
					Image: "not:::valid",
				},
			},
			imageTagOverride: "v5.0",
			wantImage:        "not:::valid",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := overrideImageTag(tt.catsrc, tt.imageTagOverride)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid image")
			} else {
				require.NoError(t, err)
				if tt.catsrc != nil {
					assert.Equal(t, tt.wantImage, tt.catsrc.Spec.Image)
				}
			}
		})
	}
}
