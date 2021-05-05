#! /bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR=$(dirname "${BASH_SOURCE[0]}")/..
SDK_VERSION=${SDK_VERSION:=v0.19.1}

# Get operator-sdk binary.
wget -O /tmp/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}/operator-sdk-${SDK_VERSION}-x86_64-linux-gnu && chmod +x /tmp/operator-sdk

pushd "${ROOT_DIR}"
OPERATOR_SDK_BIN=/tmp/operator-sdk "${ROOT_DIR}/scripts/e2e-tests.sh"
popd
