#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

: "${KUBECONFIG:?}"

# TODO(tflannag): This script likely only needs to be a wrapper around e2e-tests.sh
# and e2e-tests.sh needs to be updated to take a configurable list of operator-sdk
# test command flags.
ROOT_DIR=$(dirname "${BASH_SOURCE[0]}")/..
DEFAULTS_DIR=${ROOT_DIR}/defaults

TEST_NAMESPACE="${TEST_NAMESPACE:=openshift-marketplace}"
OPERATOR_SDK_BIN=${OPERATOR_SDK_BIN:=operator-sdk}
EXTRA_GO_TEST_FLAGS=${EXTRA_GO_TEST_FLAGS:=""}

echo "Running operator-sdk test"
${OPERATOR_SDK_BIN} version
${OPERATOR_SDK_BIN} test local \
    ./test/e2e/ \
    --debug \
    --no-setup \
    --kubeconfig "${KUBECONFIG}" \
    --operator-namespace ${TEST_NAMESPACE} \
    --up-local \
    --local-operator-flags "-defaultsDir=${DEFAULTS_DIR} -clusterOperatorName=marketplace" \
    --go-test-flags "-v -timeout 50m ${EXTRA_GO_TEST_FLAGS}" \
