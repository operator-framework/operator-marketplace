# OpenShift Marketplace - Build and Test

SHELL := /bin/bash

all: build

build: osbs-build

osbs-build:
	./build/build.sh

unit: unit-test

unit-test:
	go test -v ./pkg/...

e2e: e2e-job

e2e-job:
	go test -v -race -failfast -timeout 90m ./test/e2e/... --ginkgo.randomizeAllSpecs

install-olm-crds:
	kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.17.0/crds.yaml

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: manifests
manifests:
	./hack/update-manifests.sh

KUBE_MINOR ?= $(shell go list -m k8s.io/client-go | cut -d" " -f2 | sed 's/^v0\.\([[:digit:]]\{1,\}\)\.[[:digit:]]\{1,\}$$/1.\1/')
.PHONY: update-k8s-values  # HELP Update PSA labels with k8s version used
update-k8s-values:
	sed -i.bak -E 's/(pod-security.kubernetes.io\/enforce-version:).*/\1 "v$(KUBE_MINOR)"/' ./manifests/01_namespace.yaml
	sed -i.bak -E 's/(pod-security.kubernetes.io\/audit-version:).*/\1 "v$(KUBE_MINOR)"/' ./manifests/01_namespace.yaml
	sed -i.bak -E 's/(pod-security.kubernetes.io\/warn-version:).*/\1 "v$(KUBE_MINOR)"/' ./manifests/01_namespace.yaml
	rm ./manifests/01_namespace.yaml.bak

#SECTION Verification

.PHONY: diff
diff:
	git diff --exit-code

.PHONY: verify-update-k8s-values
verify-update-k8s-values: update-k8s-values #HELP Check if Helm Chart values are updated with k8s version
	$(MAKE) diff

.PHONY: verify #HELP Run all verification checks
verify: verify-update-k8s-values
