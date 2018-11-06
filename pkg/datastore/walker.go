package datastore

import (
	"fmt"
)

// ManifestWalker is an interface that wraps the Walk method.
//
// Walk traverses a manifest and discovers the set of CRD(s), CSV(s)
// associated with each package in the manifest.
//
// The walk method does the following:
// 	For each package
// 	  For each channel in the package
//	    a. Get the current ClusterServiceVersion (CSV) object.
//		b. Get the list of owned and required CustomResourceDefinition object(s)
//		   specified by the this CSV.
// 		c. Get the name of the older CSV that the current CSV replaces. If a
// 		   'replaces' CSV is found then repeat from b.
//
// For each package, CRD and CSV discovered, the Walk function notifies the
// given ManifestDiscovery object specified in discovery.
// On any error, the function aborts the walk operation and returns an
// appropriate error object.
type ManifestWalker interface {
	Walk(manifest *StructuredOperatorManifestData, discovery ManifestDiscovery) error
}

// ManifestDiscovery is an interface that encapsulates the notification methods
// called by a ManifestWalker while it traverses a manifest and discovers
// relevant CRD(s), CSV(s) and package(s).
type ManifestDiscovery interface {
	// NewPackage is invoked by ManifestWalker when it has completed traversing
	// all CRD(s), CSV(s) associated with a given package.
	NewPackage(operatorPackage *PackageManifest)

	// NewCSV is invoked by ManifestWalker when it comes across a
	// ClusterServiceVersion object referred to by a given channel of a package.
	NewCSV(packageName, channelName string, csv *ClusterServiceVersion)

	// NewCRD is invoked by ManifestWalker when it comes across a CRD that a
	// given ClusterServiceVersion (CSV) object owns.
	NewCRD(packageName, channelName string, crd *CustomResourceDefinition)
}

// walker implements ManifestWalker interface.
type walker struct {
}

func (w *walker) Walk(manifest *StructuredOperatorManifestData, discovery ManifestDiscovery) error {
	csvs := ClusterServiceVersionMap{}
	csvs.Load(manifest.ClusterServiceVersions)

	crds := CustomResourceDefinitionMap{}
	if err := crds.Load(manifest.CustomResourceDefinitions); err != nil {
		return err
	}

	for i, p := range manifest.Packages {
		for _, channel := range p.Channels {
			// Get the current CSV associated with the channel.
			currentCSV, ok := csvs[channel.CurrentCSVName]
			if !ok {
				return fmt.Errorf("did not find current CSV[%s] channel[%s] package[%s]", channel.CurrentCSVName, channel.Name, p.PackageName)
			}

			csv := currentCSV
			for {
				discovery.NewCSV(p.PackageName, channel.Name, csv)

				// For each CSV, we are going to get the list of CRD(s) owned
				// by it. We ignore the required list.
				owned, _, err := csv.GetCustomResourceDefintions()
				if err != nil {
					return fmt.Errorf("error getting CustomResourceDefinition of CSV[%s], channel[%s] package[%s] - %s", csv.Name, channel.Name, p.PackageName, err)
				}

				for _, key := range owned {
					crd, ok := crds[*key]
					if !ok {
						return fmt.Errorf("did not find a CRD[%s] channel[%s] package[%s]", key.Name, channel.Name, p.PackageName)
					}

					discovery.NewCRD(p.PackageName, channel.Name, crd)
				}

				// We need to walk through the chain of CSV(s) now.
				// If this CSV replaces an older version then we will take into
				// account the older CSV.
				replaces, err := csv.GetReplaces()
				if err != nil {
					return fmt.Errorf("error getting 'replaces' of CSV[%s], channel[%s] package[%s] - %s", csv.Name, channel.Name, p.PackageName, err)
				}

				// That's it, we have exhausted all the CSV(s) for this channel.
				if replaces == "" {
					break
				}

				csv, ok = csvs[replaces]
				if !ok {
					return fmt.Errorf("did not find replaces CSV[%s], channel[%s] package[%s]", replaces, channel.Name, p.PackageName)
				}
			}
		}

		discovery.NewPackage(&manifest.Packages[i])
	}

	return nil
}
