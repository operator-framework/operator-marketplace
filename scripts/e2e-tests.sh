#!/bin/bash
set -e

TEST_NAMESPACE="openshift-marketplace"
MANIFEST_FOLDER="./test/e2e/environment/"
NAMESPACED_MANIFEST="./${MANIFEST_FOLDER}/namespaced-manifest.yaml"
GLOBAL_MANIFEST="./${MANIFEST_FOLDER}/global-manifest.yaml"
OPERATOR_SOURCE_CRD="./deploy/crds/operatorsource.crd.yaml"
CATALOG_SOURCE_CONFIG_CRD="./deploy/crds/catalogsourceconfig.crd.yaml"

# Create openshift resources if they don't exist
echo "Creating openshift resources"
kubectl apply -f $OPERATOR_SOURCE_CRD
kubectl apply -f $CATALOG_SOURCE_CONFIG_CRD
if ! kubectl get namespace $TEST_NAMESPACE; then
    kubectl create namespace $TEST_NAMESPACE
fi

# Run the tests through the operator-sdk
echo "Running operator-sdk test"
operator-sdk test local ./test/e2e --up-local --kubeconfig=$KUBECONFIG --namespace $TEST_NAMESPACE
