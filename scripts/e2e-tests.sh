#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

ARG1=${1-}
if [ "$ARG1" = "minikube" ]; then
    TEST_NAMESPACE="marketplace"
else
    TEST_NAMESPACE="openshift-marketplace"
fi

OPERATOR_SDK_BIN=${OPERATOR_SDK_BIN:=operator-sdk}
export GO111MODULE=off

# Run the tests through the operator-sdk
echo "Running operator-sdk test"
${OPERATOR_SDK_BIN} --version
${OPERATOR_SDK_BIN} test local ./test/e2e/ --no-setup --go-test-flags "-v -timeout 50m" --namespace $TEST_NAMESPACE
