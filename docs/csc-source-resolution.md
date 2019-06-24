# CatalogSourceConfig Source Resolution

v2 of the `CatalogSourceConfig CRD` expects a `source` that matches the name of an `OperatorSource` on cluster that all packages must originate from. If a CatalogSourceConfig is created without specifying a `source`, the Marketplace Operator will attempt to update the `source` field with the name of a valid `OperatorSource` that contains the list of `packages`.

This document will describe how the Marketplace Operator will attempt to reconcile invalid and missing `sources` and the phase that the `CatalogSourceConfig` will eventually reach. Any `CatalogSourceConfig` placed in the `Configuring` phase can eventually be resolved if the described errors are addressed.

## CSC Includes Source Scenarios

1. If a `CatalogSourceConfig` defines a `source` that exists on the cluster and contains the requested `packages`, the `CatalogSourceConfig` will be placed in the `Succeeded` phase.
2. If a `CatalogSourceConfig` defines a `source` that exists on the cluster but does not contain the requested `packages`, the `CatalogSourceConfig` will be placed in the `Configuring` phase and the `phase.message` will be updated to reflect that the `OperatorSource` does not include the expected `packages`.
3. If a `CatalogSourceConfig` defines a `source` that does not exist on the cluster, the `CatalogSourceConfig` will be placed in the `Configuring` phase and the `phase.message` will be updated to reflect that the `source` does not exist.

## CSC Missing Source Scenarios

1. If a `CatalogSourceConfig` does not define a `source` and an `OperatorSource` contains the requested `packages`, the `CatalogSourceConfig` will be updated to include the valid `OperatorSource` as its `source` and placed in the `Succeeded` phase.
2. If a `CatalogSourceConfig` does not define a `source` and multiple `OperatorSources` contain the requested `packages`, the `CatalogSourceConfig` will be updated to include one of the valid `OperatorSources` as its `source`.
3. If a `CatalogSourceConfig` does not define a `source` and no `OperatorSource` contains the requested `packages`, the `CatalogSourceConfig` will be placed in the `Configuring` phase and the `phase.message` will be updated to reflect that a `source` could not be resolved.
