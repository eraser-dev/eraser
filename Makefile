# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/azure/eraser-manager:v0.1.0
ERASER_IMG ?= ghcr.io/azure/eraser:v0.1.0

KUSTOMIZE_VERSION ?= 3.8.9
KUBERNETES_VERSION ?= 1.23.0
ENVTEST_K8S_VERSION ?= 1.23
GOLANGCI_LINT_VERSION := v1.43.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ifdef CACHE_TO
_CACHE_TO := --cache-to $(CACHE_TO)
endif

ifdef CACHE_FROM
_CACHE_FROM := --cache-from $(CACHE_FROM)
endif

OUTPUT_TYPE ?= type=registry
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)
GO_INSTALL := ./hack/go-install.sh
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VERSION)

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint $(GOLANGCI_LINT_BIN) $(GOLANGCI_LINT_VERSION)


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Linting
.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -v

##@ Development

manifests: __controller-gen
	$(CONTROLLER_GEN) \
		crd \
		rbac:roleName=manager-role \
		webhook \
		paths="./..." \
		output:crd:artifacts:config=config/crd/bases
	rm -rf manifest_staging
	mkdir -p manifest_staging/deploy
	docker run --rm -v $(shell pwd):/eraser \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/eraser/config/default -o /eraser/manifest_staging/deploy/eraser.yaml

## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
generate: __controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

# Run e2e tests
.PHONY: e2e-test
e2e-test:
	IMAGE=${ERASER_IMG} MANAGER_IMAGE=${IMG} NODE_VERSION=kindest/node:v${KUBERNETES_VERSION} go test -count=1 -tags=e2e -v ./test/e2e

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: ## Build docker image with the manager.
	docker buildx build $(_CACHE_FROM) $(_CACHE_TO) --platform="linux/amd64" --output=$(OUTPUT_TYPE) --target manager -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

docker-build-eraser:
	docker buildx build $(_CACHE_FROM) $(_CACHE_TO) --platform="linux/amd64" --output=$(OUTPUT_TYPE) -t ${ERASER_IMG} --target eraser .

docker-push-eraser:
	docker push ${ERASER_IMG}

##@ Deployment

install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	docker run -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/crd | kubectl apply -f -

uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	docker run -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/crd | kubectl delete -f -

deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	docker run -v $(shell pwd)/config:/config -w /config/manager\
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} edit set image controller=${IMG}
	docker run -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	docker run -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/default | kubectl delete -f -

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

__controller-gen: __tooling-image
CONTROLLER_GEN=docker run -v $(shell pwd):/eraser eraser-tooling controller-gen

__tooling-image:
	docker build . \
		-t eraser-tooling \
		-f build/tooling/Dockerfile

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
