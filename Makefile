# OpenShift Marketplace - Build and Test

SHELL := /bin/bash
PKG := github.com/operator-framework/operator-marketplace/pkg
MOCKS_DIR := ./pkg/mocks
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
	./tmp/build/build.sh

unit: generate-mocks unit-test

unit-test:
	go test -v ./pkg/...

generate-mocks:
	go get github.com/golang/mock/mockgen
	
	@echo making sure directory for mocks exists
	mkdir -p $(MOCKS_DIR)

	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_datastore.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Reader=DatastoreReader,Writer=DatastoreWriter $(PKG)/datastore Reader,Writer
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_phase_reconciler.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Reconciler=PhaseReconciler,Transitioner=PhaseTransitioner $(PKG)/operatorsource/phase Reconciler,Transitioner
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_kubeclient.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Client=KubeClient $(PKG)/kube Client
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_phase_reconciler_strategy.go -package=$(OPERATORSOURCE_MOCK_PKG) $(PKG)/operatorsource PhaseReconcilerFactory
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_appregistry.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=ClientFactory=AppRegistryClientFactory,Client=AppRegistryClient $(PKG)/appregistry ClientFactory,Client

clean-mocks:
	@echo cleaning mock folder
	rm -rf $(MOCKS_DIR)

clean: clean-mocks
