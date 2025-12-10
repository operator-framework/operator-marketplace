# AGENTS.md

This document provides important context for any AI agents interacting with the operator-marketplace repository.

## Project Overview

The marketplace-operator enforces software sources for available cluster content, AKA [CatalogSources](https://github.com/operator-framework/api/blob/master/crds/operators.coreos.com_catalogsources.yaml), used by the [operator-lifecycle-manager](https://github.com/operator-framework/operator-lifecycle-manager). It is not responsible for managing the catalog runtimes on the cluster; rather it enforces the default configuration used by OLM to create these runtimes.

### APIs

The following are the key APIs used in this project:

#### CatalogSource
To manage on-cluster software sources, marketplace-operator uses the CatalogSource API. A CatalogSource defines a repository of Operators which are served via grpc by the operator-registry. For more information, see the [operator-registry repository](https://github.com/operator-framework/operator-registry).

#### ClusterOperator
In OpenShift deployments, marketplace-operator manages a ClusterOperator resource; it is responsible for updating the status of this resource to reflect status such as health and upgradeability. 

#### OperatorHub
A resource which allows users to make changes to the default software sources provided by marketplace-operator. 

### Code Structure
The codebase for marketplace-operator can be broadly categorized into the following:

#### Manager
The `main` func, found in `cmd/manager/main.go`, is the entrypoint for marketplace-operator. It can be configured by the following command-line arguments on startup:

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
|`clusterOperatorName`|string|`""`|Name of the ClusterOperator resource to provide status updates to; status updates are disabled when empty|
|`defaultsDir`|string| `""`|File path for the folder containing default CatalogSources; repository's `defaults/` folder used when empty|
|`version`|bool|`false`|Boolean flag; when provided, displays marketplace-operator source commit info then exits|
|`pprof-address`|string|`:6060`|Address to serve pprof endpoints on|
|`tls-key`|string|`""`|Path to use for private key (requires tls-cert)|
|`tls-cert`|string|`""`|Path to use for certificate (requires tls-key)|
|`leader-namespace`|string|`openshift-marketplace`|configures the namespace that will contain the leader election lock|
|`level`|string|`info`|Sets level of logger with default verbosity info level; other verbosity levels can be found [here](https://github.com/sirupsen/logrus)|

#### Controllers
Found within `pkg/controller`, marketplace-operator contains controllers for the following resource types:

| Kind | Path | Description |
|------|------|-------------|
|CatalogSource|`pkg/controller/catalogsource`|Watches and responds to any changes made to the CatalogSources defined in `defaults/`, or in the folder configured by startup flag `defaultsDir` (usage outlined under Manager section).|
|ConfigMap|`pkg/controller/configmap`|Used as a watcher for updates made to the certificate authority inside the ConfigMap `extension-apiserver-authentication`. When the CA on-disk differs from the ConfigMap's, marketplace-operator will restart to pull in the updates.|
|OperatorHub|`pkg/controller/operatorhub`|Watches the cluster for OperatorHub resources and makes any configured changes to the default CatalogSources, then updates Status of the OperatorHub to show success or failure. For instance, if the user configures the `sources` list with a CatalogSource name that doesn't exist, the OperatorHub object will show an error reflecting this.|

#### Libraries
* `pkg/certificateauthority` - Handles comparison between the certificate authority found on-disk vs on-cluster, restarting to pick up the changes.
* `pkg/client` - Wraps the raw kube client provided by operator-sdk to allow mocking in tests.
* `pkg/defaults` - Reads the default CatalogSources into memory and ensures parity with those on-cluster.
* `pkg/filemonitor` - Keeps the certs and keys used by the metrics endpoint up-to-date using `fsnotify`.
* `pkg/metrics` - Serves the prometheus metrics endpoint. Exposed via port 8383 (http) and 8081 (https).
* `pkg/operatorhub` - Handler to ensure default CatalogSource configuration in OperatorHub resource is reflected on-cluster.
* `pkg/status` - Interacts with on-cluster ClusterOperator resource to reflect marketplace-operator status and upgradeability.

### Manifests

#### Installation Manifests
Found in `manifests/`, these are the yaml files used to install marketplace-operator onto a cluster. Most notably it contains the following:
* `manifests/0000_03_marketplace-operator_02_operatorhub.cr.yaml` - An OperatorHub CR through which the default CatalogSources may be configured.
* `manifests/09_operator.yaml` - The Deployment yaml for marketplace-operator.
* `manifests/09_operator-ibm-cloud-managed.yaml` - Same as `09_operator.yaml`, but with changes required for deployment in IBM cloud managed environments.
* `manifests/04_service_account.yaml`, `manifests/05_role.yaml`, `manifests/06_role_binding.yaml` - RBAC for marketplace-operator.
* `manifests/10_clusteroperator.yaml` - The ClusterOperator object used by marketplace-operator to reflect status.

#### Default CatalogSource Manifests
By default, marketplace-operator enforces all of the CatalogSources found in the `defaults/` folder. This folder contains the following:

| Name | Path |Description |
|------|------|------------|
| redhat-operators |`defaults/01_redhat_operators.cr.yaml`| Red Hat products packaged, shipped, and supported by Red Hat |
| certified-operators |`defaults/02_certified_operators.yaml`| Products developed by software vendors outside of Red Hat; packaged and shipped in partnership with Red Hat |
| community-operators |`defaults/03_community_operators.yaml`| Unsupported community software source defined via [github repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main/operators) |
| redhat-marketplace |`defaults/04_redhat_marketplace.yaml`| ⚠ DEPRECATED: Red Hat-certified software previously available to be purchased from [Red Hat Marketplace](https://marketplace.redhat.com/sunset); no longer in use |

## Contribution Guidance

### Development Environment
Project Requirements:
* go `v1.24.6`
* container runtime such as `docker` or `podman`
* e2e: Requires access to OpenShift cluster `v4.0` or greater

### Make Targets
* `make all`, `make build`, or `make osbs-build` - builds the marketplace-operator project via `build/build.sh`
* `make unit` or `make unit-test` - run unit tests
* `make e2e` or `make e2e-job` - run e2e tests
* `make install-olm-crds` - installs CRDs required by OLM from the [operator-lifecycle-manager repository](https://github.com/operator-framework/operator-lifecycle-manager)
* `make vendor` - runs `tidy`, `vendor`, and `verify` commands for `go mod`
* `make manifests` - runs the manifest generation script at `hack/update-manifests.sh`

### Testing
* Unit tests: `go test ./pkg/...` or `make unit-test`
* E2E tests: `make e2e` or `make e2e-job` - more details can be found in `docs/e2e-testing.md`
* `go mod` verification: `make vendor`
* Testing code changes:
  * Install [Operator-SDK](https://github.com/operator-framework/operator-sdk)
  * Login to OpenShift cluster `v4.0` or greater
  * Disable CVO (ClusterVersionOperator) management of marketplace-operator:
    ```
    $ oc patch clusterversion version --type=merge -p '{"spec": {"overrides":[{"kind": "Deployment", "name": "marketplace-operator", "namespace": "openshift-marketplace", "unmanaged": true, "group": "apps"}]}}'
    ``` 
  * Delete the marketplace-operator deployment:
    ```
    $ oc delete deployment marketplace-operator -n openshift-marketplace
    ```
  * Compile the marketplace-operator and start the operator in your dev environment:
    ```
    $ operator-sdk up local --namespace=openshift-marketplace --kubeconfig=<path-to-kubeconfig-file>  
    ```

### PRs
* Fork the repository and make changes to a branch
* Run `make e2e`, `make unit`, and `make vendor`
* Squash commits to a single commit with a comprehensive summary of changes
* Push branch to forked repository
* Submit PR from fork to `operator-framework/operator-marketplace`, generally targeting `master` branch
* Ask contributors listed in `OWNERS` file for `Approved` and `lgtm` tags
* Merge PR
