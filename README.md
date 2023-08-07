# Marketplace Operator
Marketplace is a conduit to bring off-cluster operators to your cluster.

## Prerequisites
In order to deploy the Marketplace Operator, you must:
1. Have an OKD with Operator Lifecycle Manager (OLM) [installed](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md).
2. Be logged in as a user with Cluster Admin role.

## Using the Marketplace Operator

### Description
The operator manages one CRD: [OperatorHub](https://github.com/openshift/api/blob/600991d550ac9ee3afbfe994cf0889bf9805a3f5/config/v1/0000_03_marketplace-operator_01_operatorhub.crd.yaml).

#### OperatorHub

The `OperatorHub` named `cluster` is used to manage default catalogSources found on OpenShift distributions.


Here is a description of the spec fields:

- `disableAllDefaultSources` allows you to disable all the default hub sources. If this is true, a specific entry in sources can be used to enable a default source. If this is false, a specific entry in sources can be used to disable or enable a default source.
                  
- `sources` is the list of default hub sources and their configuration. If the list is empty, it implies that the default hub sources are enabled on the cluster unless disableAllDefaultSources is true. If disableAllDefaultSources is true and sources is not empty, the configuration present in sources will take precedence. The list of default hub sources and their current state will always be reflected in the status block.

Please see [here][https://docs.openshift.com/container-platform/4.13/operators/understanding/olm-understanding-operatorhub.html] for more information.

### Deploying the Marketplace Operator with OKD
The Marketplace Operator is deployed by default with OKD and no further steps are required.

## Marketplace End to End (e2e) Tests

A full writeup on Marketplace e2e testing can be found [here](docs/e2e-testing.md)

[upstream-community-operators]: deploy/upstream/07_upstream_operatorsource.cr.yaml
[community-operators]: deploy/examples/community.operatorsource.cr.yaml
