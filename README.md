# Marketplace Operator
Marketplace is a conduit to bring off-cluster operators to your cluster.

## Project Status: pre-alpha
The project is currently pre-alpha and it is expected that breaking changes to the API will be made in the upcoming releases.

## Prerequisites
In order to deploy the Marketplace Operator, you must:
1. Have an OKD or a Kubernetes cluster with Operator Lifecycle Manager (OLM) [installed](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/install/install.md).
2. Be logged in as a user with Cluster Admin role.
   * This is a stop gap measure until the RBAC permissions are defined

## Making changes to the Marketplace Operator
The Marketplace Operator is hosted publicly at `quay.io/redhat/marketplace-operator` but not all developers have push privileges on this image. If you do not have the push privilege and are developing new features for the Marketplace Operator you must build and push your Marketplace Operator image to a registry where you have push and pull privileges and update the `deploy/operator.yaml` to pull this image. The steps below outline said process:
1. Build and push your Marketplace Operator Image with the following command.
```bash
$ export REGISTRY=<SOME_REGISTRY> \
   && export NAMESPACE=<SOME_NAMESPACE> \
   && export REPOSITORY=<SOME_REPOSITORY> \
   && export TAG=<SOME_TAG> \
   && operator-sdk build $REGISTRY/$NAMESPACE/$REPOSITORY:$TAG \
   && docker push $REGISTRY/$NAMESPACE/$REPOSITORY:$TAG
```
2. Update the `deploy/operator.yaml` to pull the Marketplace Operator image you just pushed. You should update the `spec.template.spec.containers[0].image` field with the `$REGISTRY/$NAMESPACE/$REPOSITORY:$TAG` value.

## Using the Marketplace Operator

### Description
The operator manages two CRDs: [OperatorSource](./deploy/crds/operatorsource.crd.yaml) and [CatalogSourceConfig](./deploy/crds/catalogsourceconfig.crd.yaml).

`OperatorSource` is used to define the external datastore we are using to store operator bundles. At the moment we only support Quay's app-registry as our external datastore. Please see [here](deploy/examples/community.operatorsource.cr.yaml) for an example `OperatorSource`. The `endpoint` in the `spec` is typically set to `https:/quay.io/cnr` if you are using Quay's app-registry. The `registryNamespace` is the name of your app-registry namespace. `displayName` and `publisher` are optional and only needed for UI purposes. If you want an `OperatorSource` to work with private app-registry repositories, please take a look at the [Private Repo Authentication](docs/how-to-authenticate-private-repositories.md) documentation.
On adding an `OperatorSource` to an OKD cluster, operators will be visible in the [OperatorHub UI](https://github.com/openshift/console/tree/master/frontend/public/components/marketplace) in the OKD console. There is no equivalent UI in the Kubernetes console.

`CatalogSourceConfig` is used to install an operator present in the `OperatorSource` to your cluster. Behind the scenes, it will configure an OLM `CatalogSource` so that the operator can then be managed by OLM. Please see [here](deploy/examples/catalogsourceconfig.cr.yaml) for an example `CatalogSourceConfig`.
The `targetNamespace` is the namespace that OLM is watching. This is where the resulting `CatalogSource`, which will have the same name as the `CatalogSourceConfig`, is created or updated. `packages` is a comma separated list of operators. `csDisplayName` and `csPublisher` are optional but will result in the `CatalogSource` having proper UI displays. Once a `CatalogSourceConfig` is created successfully you can create a [`Subscription`](https://github.com/operator-framework/operator-lifecycle-manager#discovery-catalogs-and-automated-upgrades) for your operator referencing the newly created or updated `CatalogSource`.

Please note that the Marketplace operator uses `CatalogSourceConfigs` and `CatalogSources` internally and you will find them present in the namespace where the Marketplace operator is running. These resources can be ignored and should not be modified or used.

### Deploying the Marketplace Operator with OKD
The Marketplace Operator is deployed by default with OKD and no further steps are required.

### Deploying the Marketplace Operator with Kubernetes
First ensure that the [Operator Lifecycle Manager](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/install/install.md#install-the-latest-released-version-of-olm-for-upstream-kubernetes) is installed on your cluster.

#### Deploying the Marketplace Operator
```bash
$ kubectl apply -f deploy/upstream
```

#### Installing an operator using Marketplace
The following section assumes that Marketplace was installed in the `marketplace` namespace. For Marketplace to function you need to have at least one `OperatorSource` CR present on the cluster. To get started you can use the `OperatorSource` for [upstream-community-operators](deploy/examples/upstream.operatorsource.cr.yaml). If you are on an OKD cluster, you can skip this step as the `OperatorSource` for [community-operators](deploy/examples/community.operatorsource.cr.yaml) is installed by default instead.
```bash
$ kubectl apply -f deploy/examples/upstream.operatorsource.cr.yaml
```
Once the `OperatorSource` has been successfully deployed, you can discover the operators available using the following command:
```bash
$ kubectl get opsrc upstream-community-operators -o=custom-columns=NAME:.metadata.name,PACKAGES:.status.packages -n marketplace
NAME                           PACKAGES
upstream-community-operators   federationv2,svcat,metering,etcd,prometheus,automationbroker,templateservicebroker,cluster-logging,jaeger,descheduler
```

Now if you want to install the `descheduler` and `jaeger` operators, create a `CatalogSourceConfig` CR as shown below:
```
apiVersion: marketplace.redhat.com/v1alpha1
kind: CatalogSourceConfig
metadata:
  name: installed-upstream-community-operators
  namespace: marketplace
spec:
  targetNamespace: local-operators
  packages: descheduler,jaeger
  csDisplayName: "Upstream Community Operators"
  csPublisher: "Red Hat"
```
Note, that in the example above, `local-operators` is a namespace that OLM is watching. Deployment of this CR will cause a `CatalogSource` called `installed-upstream-community-operators` to be created in the `local-operators` namespace. This can be confirmed by `oc get catalogsource installed-upstream-community-operators -n local-operators`. Note that you can reuse the same `CatalogSourceConfig` for adding more operators.

Now you can create an OLM `Subscriptions` for `desheduler` and `jaeger`.
```
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jaeger
  namespace: local-operators
spec:
  channel: alpha
  name: jaeger
  source: installed-upstream-community-operators
  sourceNamespace: local-operators
```

## Populating your own App Registry OperatorSource

Follow the steps [here](./docs/how-to-upload-artifact.md) to upload an operator artifact to `quay.io`.

Once your operator artifact is pushed to `quay.io` you can use an `OperatorSource` to add your operator offering to Marketplace. An example `OperatorSource` is provided [here](deploy/examples/community.operatorsource.cr.yaml).

An `OperatorSource` must specify the `registryNamespace` the operator artifact was pushed to, and set the `name` and `namespace` for creating the `OperatorSource` on your cluster.

Add your `OperatorSource` to your cluster:

```bash
$ oc create -f your-operator-source.yaml
```

Once created, the Marketplace operator will use the `OperatorSource` to download your operator artifact from the app registry and display your operator offering in the Marketplace UI.

## Running End to End (e2e) Tests

To run the e2e tests defined in test/e2e that were created using the operator-sdk, first ensure that you have the following additional prerequisites:

1. The operator-sdk binary installed on your environment. You can get it by either downloading a released binary on the sdk release page [here](https://github.com/operator-framework/operator-sdk/releases/) or by pulling down the source and compiling it [locally](https://github.com/operator-framework/operator-sdk).
2. A namespace on your cluster to run the tests on, e.g.
```bash
    $ oc create namespace test-namespace
```
3. A Kubeconfig file that points to the cluster you want to run the tests on.

To run the tests, just call operator-sdk test and point to the test directory:

```bash
operator-sdk test local ./test/e2e --up-local --kubeconfig=$KUBECONFIG --namespace $TEST_NAMESPACE
```

You can also run the tests with `make e2e-test`.
