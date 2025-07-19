# OpenShift Marketplace Operator

The Operator manages the execution of cluster-wide marketplace features. Marketplace is a conduit to bring off-cluster operators to your cluster.

## Security Hardening

### Recent Changes (PR #634)

The marketplace operator now runs with `readOnlyRootFilesystem: true` as part of security hardening efforts. This change was implemented in PR #634 to enhance security posture.

### Catalog Source Security Configuration

**Important Security Note**: The default catalog sources are temporarily configured with `securityContextConfig: legacy` due to test compatibility issues. This is a temporary workaround.

**Long-term Solution Required**: 
- External tests need to be updated to expect "Permission denied" errors instead of read-only filesystem errors when testing security restrictions
- This is due to OpenShift's layered security model where user permission checks occur before filesystem read-only checks
- Once tests are updated, catalog sources should be changed back to `securityContextConfig: restricted` for optimal security

For more information about OpenShift security context constraints, see: https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html

## Installation

### Prerequisites
In order to deploy the Marketplace Operator, you must:
1. Have an OKD with Operator Lifecycle Manager (OLM) [installed](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md).
2. Be logged in as a user with Cluster Admin role.

### Using the Marketplace Operator

#### Description
The operator manages one CRD: [OperatorHub](https://github.com/openshift/api/blob/600991d550ac9ee3afbfe994cf0889bf9805a3f5/config/v1/0000_03_marketplace-operator_01_operatorhub.crd.yaml).

#### OperatorHub
OperatorHub is used to change the state of the default CatalogSources provided with Marketplace. For example, setting `spec.disableAllDefaultSources` to true will disable all the default CatalogSources.

To restore defaults, an empty OperatorHub resource can be applied against the cluster:
```bash
$ oc apply -f - <<EOF
apiVersion: config.openshift.io/v1
kind: OperatorHub
metadata:
  name: cluster
spec: {}
EOF
```

#### Configuration
The OperatorHub CustomResource is cluster scoped. It contains a `sources` list that can be used to enable/disable default catalog sources. Currently these are:
- redhat-operators (enabled by default)
- certified-operators (enabled by default)  
- redhat-marketplace (enabled by default)
- community-operators (enabled by default)

Custom CatalogSources and community CatalogSources should never be added to this list as they might get cleaned up or modified by the cluster. In some cases, admins may want to disable the community-operators CatalogSource due to lack of official support.

Marketplace can also be configured by creating an `OperatorHub` object which allows for configuration of the default CatalogSources that are deployed by default with the operator.

#### Example
```yaml
apiVersion: config.openshift.io/v1
kind: OperatorHub
metadata:
  name: cluster
spec:
  disableAllDefaultSources: true # Disables all sources
  sources: # List of sources and their individual enablement status
  - disabled: false
    name: community-operators
```

## Development

### Prerequisites
- git
- mercurial 3.9+
- [operator-sdk][operator_sdk]
- [dep][dep_tool] v0.5.0+
- [go][go_tool] v1.13+
- [docker][docker_tool] v17.03+
- Access to a Kubernetes v1.11.3+ cluster.

### Download Repository
```
$ git clone https://github.com/operator-framework/operator-marketplace
$ cd operator-marketplace
```

### Build
```
$ make build
```

### Test
```
$ make test
```

#### End to End (e2e) tests

##### Prerequisites

You must have a running Kubernetes or OpenShift cluster to run the tests against.

*Note*: If you are running the tests against a local OpenShift 4.0 cluster created by `openshift-install`, you may need to run the following command to give yourself cluster-admin privileges:

```bash
$ oc adm policy add-cluster-role-to-user cluster-admin <your-user-name>
```

##### Running the e2e tests

```
$ make test-e2e
```

## Release

Marketplace operator release process is driven by OpenShift's ART team. They build operator images on demand and update [image references in this file](https://github.com/openshift/cluster-version-operator/blob/master/install/0000_90_cluster-version-operator_03_deployment.yaml).

## Contributing
Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## License
Marketplace Operator is under Apache 2.0 license. See the [LICENSE](LICENSE) file for details.

[dep_tool]:https://golang.github.io/dep/docs/installation.html
[go_tool]:https://golang.org/dl/
[docker_tool]:https://docs.docker.com/install/
[operator_sdk]:https://github.com/operator-framework/operator-sdk 