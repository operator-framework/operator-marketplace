# Marketplace Operator
Marketplace is a conduit to bring off-cluster operators to your cluster.

## Prerequisites
In order to deploy the Marketplace Operator, you must:
1. Have an OKD or a Kubernetes cluster with Operator Lifecycle Manager (OLM) [installed](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md).
2. Be logged in as a user with Cluster Admin role.

## Using the Marketplace Operator

### Description
The operator manages a set of [default CatalogSources](./defaults). If these CatalogSources are modified or deleted, the operator recreates them.

#### CatalogSource

A `CatalogSource` acts as a repository of operator bundles, which are collections of operator metadata including CSVs, CRDs, package definitions etc. New operators can be made available either by adding their bundles to the [community-operators](https://github.com/operator-framework/community-operators) repository or by creating a custom registry image with a `CatalogSource` referencing it.

Here is a description of the spec fields:

- `priority` determines the order in which `CatalogSources` are queried for package resolution. A higher priority `CatalogSource` is preferred over a lower priority one during dependency resolution. If two `CatalogSources` have the same priority, then they will be ordered lexicographically based on their names. By default, a new `CatalogSource` has priority set to 0, and all default `CatalogSources` have negative priorities.

- `updateStrategy` is used to determine the frequency at which the source image is polled for `grpc` type `CatalogSources`. The update takes some time to complete, so the `interval` should not be too short. An interval of `10m` - `15m` should be sufficient for this.

- `secrets` are a list of secrets used to access contents of the catalog. These are tried for every catalog entry, so this list should be kept short.

- `sourceType` specifies the data source type that the catalog source references. Supported `sourceTypes` include `"grpc"` and `"configmap"`. The recommended source type is `"grpc"`.

- `image` is the registry image that is queried for `grpc` type `CatalogSources`.

- `address` is specified as \<host or ip>:\<port> and can be used to connect to a pre-existing registry for `grpc` type `CatalogSources`. This field is ignored if the `image` field is non-empty.

- `configMap` is used in `configmap` type `CatalogSources` to refer to the `ConfigMap` that backs the registry.

- `displayName`, `description`, `icon` and `publisher` are optional and only needed for UI purposes.

Please see [here][community-operators] for an example `CatalogSource`.

On adding a `CatalogSource` to an OKD cluster, operators will be visible in the [OperatorHub UI](https://github.com/openshift/console/tree/master/frontend/public/components/operator-hub) in the OKD console. There is no equivalent UI in the Kubernetes console.

Once a `CatalogSource` is created successfully you can create a [`Subscription`](https://olm.operatorframework.io/docs/tasks/install-operator-with-olm/) for your operator referencing the newly created or updated `CatalogSource`.

### Deploying the Marketplace Operator with OKD
The Marketplace Operator is deployed by default with OKD and no further steps are required.

### Deploying the Marketplace Operator with Kubernetes
First ensure that the [Operator Lifecycle Manager](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/install/install.md#install-the-latest-released-version-of-olm-for-upstream-kubernetes) is installed on your cluster.

#### Deploying the Marketplace Operator
```bash
$ kubectl apply -f deploy/upstream
```

#### Installing an operator using Marketplace

The following section assumes that Marketplace was installed in the `marketplace` namespace. To discover operators, you need at least one `CatalogSource` CR present on the cluster. To get started, you can use the [community-operators] `CatalogSource`. An OKD cluster will have the [default `CatalogSources`](./defaults) installed, so you can skip this step.

```bash
$ kubectl apply -f deploy/examples/community.catalogsource.cr.yaml
```

You can also [create a registry image with custom operators](https://olm.operatorframework.io/docs/tasks/make-operator-part-of-catalog/) for your `CatalogSource` to reference.

Once the `CatalogSource` has been successfully deployed, you can discover the operators available using the following command:
```bash
$ kubectl get packagemanifests
NAME                           PACKAGES
upstream-community-operators   federationv2,svcat,metering,etcd,prometheus,automationbroker,templateservicebroker,cluster-logging,jaeger,descheduler
```

Now if you want to install the `descheduler` and `jaeger` operators, create OLM [`Subscriptions`](https://github.com/operator-framework/operator-lifecycle-manager/tree/274df58592c2ffd1d8ea56156c73c7746f57efc0#discovery-catalogs-and-automated-upgrades) for `desheduler` and `jaeger` in the appropriate namespace. Depending on the `InstallModes` allowed on the operator CSVs, this may be one or more namespaces watched by OLM.

```
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jaeger
  namespace: marketplace
spec:
  channel: alpha
  name: jaeger
  source: upstream-community-operators
  sourceNamespace: marketplace
  installPlanApproval: Automatic
```

For OLM to act on your subscription please note that the `InstallMode(s)` present on your `CSV` must be compatible with the 
an [`OperatorGroup`] that matches the [`InstallMode(s)`](https://github.com/operator-framework/operator-lifecycle-manager/blob/274df58592c2ffd1d8ea56156c73c7746f57efc0/Documentation/design/building-your-csv.md#operator-metadata) in your [`CSV`](https://github.com/operator-framework/operator-lifecycle-manager/blob/274df58592c2ffd1d8ea56156c73c7746f57efc0/Documentation/design/building-your-csv.md#what-is-a-cluster-service-version-csv) needs to be present in the subscription namespace (which is `marketplace` in this example).

For OKD, the `openshift-marketplace` namespace is the global catalog namespace, so a subscription to an operator from a `CatalogSource` in the `openshift-marketplace` namespace can be created in any namespace.

#### Uninstalling an operator via the CLI

After an operator has been installed, to uninstall the operator you need to delete the following resources. Below we uninstall the `jaeger` operator as an example.

Delete the `Subscription` in the namespace that the operator was installed into. For upstream Kubernetes, this is the `marketplace` namespace. Keeping to the above example subscription `jaeger`, we can run the following command to delete it from the command line:

```bash
$ kubectl delete subscription jaeger -n marketplace
```

For OKD, if the install was done via the OpenShift OperatorHub UI, the subscription will be named after the operator's packageName and will be located in the namespace you chose in the UI. By modifying the namespace in the above command it can be used to delete the appropriate subscription.

Delete the `ClusterServiceVersion` in the namespace that the operator was installed into. This will also delete the operator deployment, pod(s), rbac, and other resources that OLM created for the operator. This also deletes any corresponding CSVs that OLM "Copied" into other namespaces watched by the operator.

```bash
$ kubectl delete clusterserviceversion jaeger-operator.v1.8.2 -n marketplace
```

## Populating your own CatalogSource Image

Follow the steps [here](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#push-to-quayio) to upload operator artifacts to `quay.io`.

Once your operator artifact is pushed to `quay.io` you can create an index image using [`opm`](https://github.com/operator-framework/operator-registry#building-an-index-of-operators-using-opm). Then, this registry index image can be used in the `CatalogSource`.

The `CatalogSource` priority decides how operator dependencies get resolved, so ensure that your `CatalogSources` have a high enough priority to be used first. The default priority of 0 is higher than any of the ones provided by default, so any custom `CatalogSource` will have precedence over the [default CatalogSources](./defaults) without any additional configuration.

Add your `CatalogSource` to your cluster:

```bash
$ oc create -f your-operator-source.yaml
```

Once created, the Marketplace operator will use the `CatalogSource` to download your operator artifact from the app registry and display your operator offering in the Marketplace UI.

You can also access private AppRegistry repositories via an authenticated `CatalogSource`, which you can learn more about [here](docs/how-to-authenticate-private-repositories.md).

## Marketplace End to End (e2e) Tests

A full writeup on Marketplace e2e testing can be found [here](docs/e2e-testing.md)

[upstream-community-operators]: deploy/upstream/07_upstream_operatorsource.cr.yaml
[community-operators]: deploy/examples/community.catalogsource.cr.yaml

## Enabling/Disabling default CatalogSources on OpenShift and OKD

By default, the `marketplace-operator` manages a set of [`default CatalogSources`](../defaults). This means that unlike user defined `CatalogSources`, these get recreated if they are deleted or modified. In order to remove one or more of these `CatalogSources` from the cluster, you can make use of the `OperatorHub` resource present on OpenShift.

By default, OpenShift has a cluster level `OperatorHub` resource with the name `cluster` which the `marketplace-operator` uses to manage its default `CatalogSources`. Modifying this custom resource will allow you selectively delete some or all of these default `CatalogSources`

The `OperatorHub` spec has the following fields of interest
- `sources`: This is a list of CatalogSources specifying `name` and whether they are `disabled`, you can apply 
- `disableAllDefaultSources`: This indicates the default action for the managed `CatlogSources` not present in the `sources` list.

For instance, to enable only Red Hat operators to be discovered on the `OperatorHub` UI, you can update the `OperatorHub` CR as follows:
```
(
cat <<EOF
apiVersion: config.openshift.io/v1
kind: OperatorHub
metadata:
  name: cluster
spec:
  disableAllDefaultSources: true
  sources:
    - name: "redhat-operators"
      disabled: false
    - name: "redhat-marketplace"
      disabled: false
EOF
 ) | oc apply -f -
```

This will delete both the `certified-operators` and `community-operators` `CatalogSources` from the cluster.

If you have the pull-secret for `registry.redhat.io`, [you can also enable Red Hat operators on OKD](https://github.com/openshift/okd/blob/0975f9bc5f472e33a62cdfafac8cb664848eb7ce/FAQ.md#how-can-i-enable-the-non-community-red-hat-operators)
