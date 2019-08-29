package status

const (
	// AppRegistryMetadataEmptyError captures when OperatorSource endpoint returns
	// an empty manifest list while reconciling an enabled default OperatorSource.
	AppRegistryMetadataEmptyError string = "AppRegistryMetadataEmptyError"

	// AppRegistryOptionsError captures when there is an error building an AppRegistry
	// Options object while reconciling an enabled default OperatorSource.
	AppRegistryOptionsError string = "AppRegistryOptionsError"

	// AppRegistryFactoryError captures when there is an error creating a new AppRegistry
	// client using the AppRegistryFactory while reconciling an enabled default OperatorSource.
	AppRegistryFactoryError string = "AppRegistryFactoryError"

	// AppRegistryListPackagesError captures when there is an error returned by the AppRegistry
	// client ListPackages function while reconciling an enabled default OperatorSource.
	AppRegistryListPackagesError string = "AppRegistryListPackagesError"

	// DataStoreWriteError captures when there is an error writing Operator metadata to the
	// datastore while reconciling an enabled default OperatorSource.
	DataStoreWriteError string = "DataStoreWriteError"

	// EnsureResourcesError captures when there is an error ensuring that all GRPC resources
	// exist.
	EnsureResourcesError string = "EnsureResourcesError"

	// DefaultError captures when there is an unknown error. This is a default value returned
	// when the cause for a failing enabled default OperatorSource is unknown.
	DefaultError string = ""
)
