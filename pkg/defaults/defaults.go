package defaults

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"

	semver "github.com/blang/semver/v4"
	"github.com/containers/image/docker/reference"
)

var (
	// Dir is the directory where the default CatalogSources definitions are
	// placed on disk. It will be empty if defaults are not required.
	Dir string

	// globalCatsrcDefinitions is used to keep an in-memory record of default
	// CatalogSources as found on disk. It is a map of CatalogSource name
	// to the CatalogSource definition in the defaults directory. It is
	// populated just once during runtime to prevent new defaults from being
	// injected into the operator image.
	globalCatsrcDefinitions = make(map[string]olmv1alpha1.CatalogSource)

	// defaultConfig is the default configuration for the cluster in the absence
	// of a an OperatorHub config object or if there is one with an empty spec.
	// The default is for all the CatalogSources in the globalDefinitions to be
	// enabled.
	defaultConfig = make(map[string]bool)
)

const (
	defaultCatsrcAnnotationKey   string = "operatorframework.io/managed-by"
	defaultCatsrcAnnotationValue string = "marketplace-operator"
	defaultCatsrcVersionString   string = "0.0.1-snapshot"
)

// Defaults is the interface that can be used to ensure the default set
// of CatalogSource resources are always present on cluster.
type Defaults interface {
	EnsureAll(ctx context.Context, client wrapper.Client) map[string]error
	Ensure(ctx context.Context, client wrapper.Client, sourceName string) error
}

type defaults struct {
	catsrcDefinitions map[string]olmv1alpha1.CatalogSource
	config            map[string]bool
}

// New returns an instance of defaults
func New(catsrcDefinitions map[string]olmv1alpha1.CatalogSource, config map[string]bool) Defaults {
	// Doing this to remove the need for checking at calls sites. This can be
	// made to return an error if error checking at calls sites is preferable.
	if catsrcDefinitions == nil || config == nil {
		panic("Defaults cannot be initialized with nil definitions or config")
	}
	return &defaults{
		catsrcDefinitions: catsrcDefinitions,
		config:            config,
	}
}

// Ensure checks if the given CatalogSource source is one of the
// defaults and if it is, it ensures it is present or absent on the cluster
// based on the config.
func (d *defaults) Ensure(ctx context.Context, client wrapper.Client, sourceName string) error {
	catsrc, present := d.catsrcDefinitions[sourceName]
	if !present {
		return nil
	}
	return ensureCatsrc(ctx, client, d.config, catsrc)
}

// EnsureAll processes all the default Catalogsources and ensures they are present
// or absent on the cluster based on the config.
func (d *defaults) EnsureAll(ctx context.Context, client wrapper.Client) map[string]error {
	result := make(map[string]error)
	for name := range d.config {
		err := d.Ensure(ctx, client, name)
		if err != nil {
			result[name] = err
		}
	}
	return result
}

// GetGlobals returns the global CatalogSource definitions and the
// default config
func GetGlobals() (map[string]olmv1alpha1.CatalogSource, map[string]bool) {
	return globalCatsrcDefinitions, defaultConfig
}

// GetGlobalCatalogSourceDefinitions returns the global CatalogSource definitions
func GetGlobalCatalogSourceDefinitions() map[string]olmv1alpha1.CatalogSource {
	return globalCatsrcDefinitions
}

// GetDefaultConfig returns the global OperatorHub configuration
func GetDefaultConfig() map[string]bool {
	return defaultConfig
}

// IsDefaultSource returns true if the given name is one of the default
// CatalogSources
func IsDefaultSource(name string) bool {
	_, present := defaultConfig[name]
	return present

}

// PopulateGlobals populates the global definitions and default config. If Dir
// is blank, the global definitions and config will be initialized but empty.
// imageTagOverride updates the image tags for the default catalogSources with
// the given non-empty tag.
func PopulateGlobals(imageTagOverride string) error {
	var err error
	globalCatsrcDefinitions, defaultConfig, err = populateDefsConfig(Dir, imageTagOverride)
	return err
}

// populateDefsConfig returns populated CatalogSource definitions from files present
// in the @dir directory and an enabled config. It returns error on the first
// issue it runs into. The function also guarantees to return an empty map on error.
func populateDefsConfig(dir, imageTagOverride string) (map[string]olmv1alpha1.CatalogSource, map[string]bool, error) {
	catsrcDefinitions := make(map[string]olmv1alpha1.CatalogSource)
	config := make(map[string]bool)
	// Default directory has not been specified
	if dir == "" {
		return catsrcDefinitions, config, nil
	}

	_, err := os.Stat(dir)
	if err != nil {
		return catsrcDefinitions, config, err
	}

	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return catsrcDefinitions, config, err
	}

	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		catsrc, err := getCatsrcDefinition(fileName)
		if err != nil {
			// Reinitialize the definitions as we hard error on even one failure
			catsrcDefinitions = make(map[string]olmv1alpha1.CatalogSource)
			config = make(map[string]bool)
			return catsrcDefinitions, config, err
		}

		// Override image tags with ones matching OpenShift <major>.<minor> for
		// default CatalogSources
		if err = overrideImageTag(catsrc, imageTagOverride); err != nil {
			return map[string]olmv1alpha1.CatalogSource{}, map[string]bool{},
				fmt.Errorf("unable to update image tags for default CatalogSource %s: %w", catsrc.Name, err)
		}

		catsrcDefinitions[catsrc.Name] = *catsrc
		config[catsrc.Name] = false
	}
	return catsrcDefinitions, config, nil
}

// GetCatalogSourceImageTagOverride returns a tag of the form `v<major>.<minor>`
// where <major> and <minor> are the major and minor version parts of the semver
// argument provided through versionString, provided the version string has a
// major version of 4. This is used for determining what image tag to use on
// a default CatalogSource based on the OCP version of the cluster it is running on,
// given the 5.0 catalogsources will be shipped to both 4.23 and 5.0 clusters.
// This may be removed in 5.1+
func GetCatalogSourceImageTagOverride(versionString string) (string, error) {
	// Return empty if not in OpenShift or version is default/unknown
	if len(versionString) == 0 || versionString == defaultCatsrcVersionString {
		return "", nil
	}

	v, err := semver.Parse(versionString)
	if err != nil {
		return "", fmt.Errorf("failed to parse version string %q: %w", versionString, err)
	}

	// Only override for 4.x OpenShift versions
	if v.Major != 4 {
		return "", nil
	}

	return fmt.Sprintf("v%d.%d", v.Major, v.Minor), nil
}

// overrideImageTag overrides the tag for a given CatalogSource's image with
// a tag exactly matching `v5.0`, provided the CatalogSource has a non-empty image field
// The image tag override only applies to non-digest based images. If called on a
// CatalogSource with a digest based image, the image remains unchanged.
func overrideImageTag(catsrc *olmv1alpha1.CatalogSource, imageTagOverride string) error {
	if len(imageTagOverride) == 0 {
		return nil
	}
	if catsrc == nil {
		return nil
	}

	// Do not override image tags for non-image based CatalogSources
	if len(catsrc.Spec.Image) == 0 {
		return nil
	}

	catsrcRef, err := reference.ParseNormalizedNamed(catsrc.Spec.Image)
	if err != nil {
		return fmt.Errorf("invalid image %s for CatalogSource %s: %w", catsrc.Spec.Image, catsrc.Name, err)
	}

	// Skip digest-based images - the default behavior of Canonical references
	// when converted to string is to ignore tags in favor of digests
	if _, ok := catsrcRef.(reference.Canonical); ok {
		return nil
	}

	// Tag substitution should only happen on v5.0 images
	if taggedRef, ok := catsrcRef.(reference.Tagged); !ok || taggedRef.Tag() != "v5.0" {
		return nil
	}

	// Override reference tag
	taggedRef, err := reference.WithTag(catsrcRef, imageTagOverride)
	if err != nil {
		return fmt.Errorf("unable to update tag on image %s to %s: %w", catsrc.Spec.Image, imageTagOverride, err)
	}

	catsrc.Spec.Image = taggedRef.String()
	return nil
}
