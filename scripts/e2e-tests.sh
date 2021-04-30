#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

: "${KUBECONFIG:?}"

TEST_NAMESPACE="${TEST_NAMESPACE:=openshift-marketplace}"
OPERATOR_SDK_BIN=${OPERATOR_SDK_BIN:=operator-sdk}
EXTRA_GO_TEST_FLAGS=${EXTRA_GO_TEST_FLAGS:=""}

echo "Running the e2e testing suite"
${OPERATOR_SDK_BIN} version
${OPERATOR_SDK_BIN} test local \
    ./test/e2e/ \
    --debug \
    --no-setup \
    --kubeconfig "${KUBECONFIG}" \
    --operator-namespace ${TEST_NAMESPACE} \
    --go-test-flags "-v -timeout 50m ${EXTRA_GO_TEST_FLAGS}"
