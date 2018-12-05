# Marketplace Operator

## Project Status: pre-alpha
The project is currently pre-alpha and it is expected that breaking changes to the API will be made in the upcoming releases.

## Prerequisites
In order to deploy the Marketplace Operator, you must:
1. Have an OKD or a Kubernetes cluster with Operator Lifecycle Manager [installed](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/install/install.md).
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

The marketplace operator manages two CRDs: [OperatorSource](./deploy/operatorsource.crd.yaml) and [CatalogSourceConfig](./deploy/catalogsourceconfig.crd.yaml). When an OperatorSource CR is created in the same namespace as where the marketplace operator is running (we recommend the namespace be called "marketplace"), the operator will download artifacts stored in the registry specified in this OperatorSource CR (for now, please see documentation about using [quay](https://quay.io)'s appregistry API). For an example of this OperatorSource CR please see the [examples](./deploy/examples/) folder.

The operator will then create a CatalogSourceConfig CR which will, for the time being, trigger the marketplace operator to create a ConfigMap CR and CatalogSource CR. The package-server, managed by [OLM](https://github.com/operator-framework/operator-lifecycle-manager), will then respond to the creation of these CRs and allow the external operators to be visible in the [marketplace UI](https://github.com/openshift/console/tree/master/frontend/public/components/marketplace).

### Deploying the Marketplace Operator with OKD
It is important to note that the order in which you apply the deployment files matters, do not execute the `oc apply` commands featured in this section out of order.

#### Deploying the Marketplace Operator
```bash
$ oc apply -f deploy/catalogsourceconfig.crd.yaml
$ oc apply -f deploy/operatorsource.crd.yaml
$ oc apply -f deploy/service_account.yaml
$ oc apply -f deploy/role.yaml
$ oc apply -f deploy/role_binding.yaml
$ oc apply -f deploy/operator.yaml
```

#### Deploying the Marketplace Operator with OLM
```bash
$ oc apply -f deploy/catalogsourceconfig.crd.yaml
$ oc apply -f deploy/operatorsource.crd.yaml
$ oc apply -f deploy/service_account.yaml
$ oc apply -f deploy/role.yaml
$ oc apply -f deploy/role_binding.yaml
$ oc apply -f deploy/marketplace.csv.yaml
```

### Deploying the Marketplace Operator with Kubernetes
Execute the commands found in the [OKD section](#deploying-the-marketplace-operator-with-okd) in the same order subsituting `kubectl` for `oc`.

Note that a Kubernetes cluster does not have OLM deployed by default.

## Populating your own App Registry OperatorSource

Follow the steps [here](./docs/how-to-upload-artifact.md) to upload an operator artifact to `quay.io`.

Once your operator artifact is pushed to `quay.io` you can use an `OperatorSource` to add your operator offering to Marketplace. An example `OperatorSource` is provided [here](deploy/examples/operatorsource.cr.yaml).

An `OperatorSource` must specify the `registryNamespace` the operator artifact was pushed to, and set the `name` and `namespace` for creating the `OperatorSource` on your cluster.

Add your `OperatorSource` to your cluster:

```bash
$ oc create -f your-operator-source.yaml
```

Once created, the Marketplace operator will use the `OperatorSource` to download your operator artifact from the app registry and display your operator offering in the Marketplace UI.
