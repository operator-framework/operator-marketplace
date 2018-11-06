package datastore

// ClusterServiceVersionMap is a map of ClusterServiceVersion object(s).
// The name of the ClusterServiceVersion object is used as the key.
type ClusterServiceVersionMap map[string]*ClusterServiceVersion

// Load accepts a list of ClusterServiceVersion object(s) and loads each one
// into the map.
func (csvs ClusterServiceVersionMap) Load(list []ClusterServiceVersion) {
	for i, csv := range list {
		csvs[csv.Name] = &list[i]
	}
}

// Values returns a list of all ClusterServiceVersion object(s)
// stored in the map.
func (csvs ClusterServiceVersionMap) Values() []*ClusterServiceVersion {
	if len(csvs) == 0 {
		return nil
	}

	values := make([]*ClusterServiceVersion, 0, len(csvs))
	for _, v := range csvs {
		values = append(values, v)
	}

	return values
}
