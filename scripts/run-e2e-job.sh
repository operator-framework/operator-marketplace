set -o errexit
set -o nounset
set -o pipefail

MARKETPLACE_OPERATOR_ROOT=$(dirname "${BASH_SOURCE}")/..
SDK_VERSION=v0.2.0
KUBE_VERSION=v1.11.3

# Get operator-sdk binary.
wget -O /tmp/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}/operator-sdk-${SDK_VERSION}-x86_64-linux-gnu && chmod +x /tmp/operator-sdk
wget -O /tmp/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubectl && chmod +x /tmp/kubectl 
cd $MARKETPLACE_OPERATOR_ROOT
. ./scripts/e2e-tests.sh
