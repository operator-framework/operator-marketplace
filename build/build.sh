#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

GIT_COMMIT=${SOURCE_GIT_COMMIT:-$(git rev-parse HEAD)}

BIN_DIR="$(pwd)/build/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="marketplace-operator"
REPO_PATH="github.com/operator-framework/operator-marketplace/"
BUILD_PATH="${REPO_PATH}/cmd/manager"
echo "building "${PROJECT_NAME}"..."
CGO_ENABLED=1 CGO_DEBUG=1 GO111MODULE=off go build -ldflags "-X '${REPO_PATH}pkg/version.GitCommit=${GIT_COMMIT}'" -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
