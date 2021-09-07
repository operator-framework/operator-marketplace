#! /bin/bash

OPENAPI_GEN="go run k8s.io/kube-openapi/cmd/openapi-gen/"

echo "Generating the v1 openapi schema"
${OPENAPI_GEN} --logtostderr=true \
                  -i ./pkg/apis/operators/v1/ \
                  -o "" \
                  -O zz_generated.openapi \
                  -p ./pkg/apis/operators/v1/ \
                  -h ./hack/boilerplate.go.txt \
                  -r "-"

echo "Generating the v2 openapi schema"
${OPENAPI_GEN} --logtostderr=true \
                  -i ./pkg/apis/operators/v2/ \
                  -o "" \
                  -O zz_generated.openapi \
                  -p ./pkg/apis/operators/v2/ \
                  -h ./hack/boilerplate.go.txt \
                  -r "-"
