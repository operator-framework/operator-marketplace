package datastore

func newDecomposer() *decomposer {
	return &decomposer{
		packages:         make([]*SingleOperatorManifest, 0),
		csvMapForPackage: ClusterServiceVersionMap{},
		crdMapForPackage: CustomResourceDefinitionMap{},
	}
}

// decomposer implements ManifestWalker interface. It uses the notification
// methods invoked by ManifestWalker to decompose a multi-operator manifest into
// a set of single-operator manifest(s).
//
// For example, if a manifest specifies multiple operator(s) (like "etcd",
// "prometheus", "amq") this function will decompose it into a set of
// single-operator manifest(s), one for each operator mentioned above.
//
// Each individual operator manifest has a package section with a set of
// channel(s) and a list of CRD(s) and CSV(s) that this operator manages.
type decomposer struct {
	packages         []*SingleOperatorManifest
	csvMapForPackage ClusterServiceVersionMap
	crdMapForPackage CustomResourceDefinitionMap
}

// Packages returns the list of operator package(s) bundled by the
// decomposer.
//
// If no operator package has been added by the decomposer,
// it returns an empty list.
func (d *decomposer) Packages() []*SingleOperatorManifest {
	return d.packages
}

func (d *decomposer) NewPackage(operatorPackage *PackageManifest) {
	defer func() {
		d.csvMapForPackage = ClusterServiceVersionMap{}
		d.crdMapForPackage = CustomResourceDefinitionMap{}
	}()

	pkg := &SingleOperatorManifest{
		Package:                   operatorPackage,
		ClusterServiceVersions:    d.csvMapForPackage.Values(),
		CustomResourceDefinitions: d.crdMapForPackage.Values(),
	}

	d.packages = append(d.packages, pkg)
}

func (d *decomposer) NewCSV(packageName, channelName string, csv *ClusterServiceVersion) {
	// Ignore this CSV if we have already seen a CSV by this name.
	if _, ok := d.csvMapForPackage[csv.Name]; ok {
		return
	}

	d.csvMapForPackage[csv.Name] = csv
}

func (d *decomposer) NewCRD(packageName, channelName string, crd *CustomResourceDefinition) {
	// Ignore this CRD if we have already seen a CRD by this name.
	if _, ok := d.crdMapForPackage[crd.Key()]; ok {
		return
	}

	d.crdMapForPackage[crd.Key()] = crd
}
