# OpenShift Marketplace - Build and Test

SHELL := /bin/bash
PKG := github.com/operator-framework/operator-marketplace/pkg
MOCKS_DIR := ./pkg/mocks
CONTROLLER_RUNTIME_PKG := sigs.k8s.io/controller-runtime/pkg
OPERATORSOURCE_MOCK_PKG := operatorsource_mocks

# If the GOBIN environment variable is set, 'go install' will install the 
# commands to the directory it names, otherwise it will default of $GOPATH/bin.
# GOBIN must be an absolute path.
ifeq ($(GOBIN),)
mockgen := $(GOPATH)/bin/mockgen
else
mockgen := $(GOBIN)/mockgen
endif

all: osbs-build

osbs-build:
	# hack/build.sh
	./build/build.sh

unit: unit-test

unit-test:
	GO111MODULE=off go test -v ./pkg/...

e2e-test:
	./scripts/e2e-tests.sh

e2e-job:
	./scripts/run-e2e-job.sh

e2e-test-minikube:
	./scripts/e2e-tests.sh minikube
