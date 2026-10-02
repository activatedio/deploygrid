# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: generate
generate: controller-gen
	go generate gen.go
	$(CONTROLLER_GEN) crd paths="./pkg/apis/..." output:crd:artifacts:config=crds
	go run ./hack/crdgen --chart-name=deploygrid ./crds > ./charts/deploygrid/templates/crds.yaml
	$(MAKE) fmt


fmt:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/daixiang0/gci@latest
	go fmt ./...
	goimports -w .
	gci -w .


generate_mocks:
	find . -type f | grep mock_ | xargs rm || true
	go install github.com/vektra/mockery/v2@v2.46.3
	mockery

dev_containers:
	docker compose stop
	docker compose rm -f
	docker compose up -d

dev_kind:
	./kind/teardown.sh || true
	./kind/setup.sh

test:
	DEPLOYGRID_LOGGING_LEVEL=info go test ./...

lint: golangci-lint
	$(GOLANGCI_LINT) run ./...

serve:
	DEPLOYGRID_SWAGGER_SWAGGER_UI_URL=http://127.0.0.1:8081 CONFIG_PATH=./testdata/config.yaml DEPLOYGRID_LOGGING_DEV_MODE=true go run ./cmd/main

## Run a collector against kind-app-cluster-2, pushing to the local server
dev_collector:
	DEPLOYGRID_COLLECTOR_SERVER=http://127.0.0.1:8080/api \
	DEPLOYGRID_COLLECTOR_CLUSTER=kind-app-cluster-2 \
	DEPLOYGRID_COLLECTOR_TOKEN=$$(kubectl --context kind-ops-cluster-1 get secret kind-app-cluster-2-collector-token -n deploygrid -o jsonpath='{.data.token}' | base64 -d) \
	DEPLOYGRID_COLLECTOR_KUBE_CONFIG_PATH=.kind/kubeconfig-app-cluster-2.yaml \
	DEPLOYGRID_LOGGING_DEV_MODE=true go run ./cmd/main collector

## Apply CRDs and sample custom resources to the kind ops cluster
dev_kind_crs:
	kubectl --context kind-ops-cluster-1 apply -f ./crds
	kubectl --context kind-ops-cluster-1 apply -f ./kind/systems.yaml

##@ Build

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.16.4
GOLANGCI_LINT_VERSION ?= v2.2.2

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))


# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
