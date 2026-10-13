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
	kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.32.0/crds.yaml

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: manifests
manifests:
	./hack/update-manifests.sh
