VERSION := v1.0.0-beta.2

# Image URL to use all building/pushing image targets
TRIVY_SCANNER_IMG ?= ghcr.io/azure/eraser-trivy-scanner:${VERSION}
MANAGER_IMG ?= ghcr.io/azure/eraser-manager:${VERSION}
ERASER_IMG ?= ghcr.io/azure/eraser:${VERSION}
COLLECTOR_IMG ?= ghcr.io/azure/collector:${VERSION}
VULNERABLE_IMG ?= docker.io/library/alpine:3.7.3
NON_VULNERABLE_IMG ?= ghcr.io/azure/non-vulnerable:latest
E2E_TESTS ?= $(shell find ./test/e2e/tests/ -mindepth 1 -type d)
TEST_LOGDIR ?= $(PWD)/test_logs

KUSTOMIZE_VERSION ?= 3.8.9
KUBERNETES_VERSION ?= 1.25.3
NODE_VERSION ?= 16-bullseye-slim
ENVTEST_K8S_VERSION ?= 1.25
GOLANGCI_LINT_VERSION := 1.43.0
TRIVY_VERSION ?= $(shell go list -f '{{ .Version }}' -m github.com/aquasecurity/trivy)

PLATFORM ?= linux

# build variables
LDFLAGS ?= $(shell build/version.sh "${VERSION}")
ERASER_LDFLAGS ?= -extldflags=-static $(LDFLAGS) -w
TRIVY_SCANNER_LDFLAGS ?= $(ERASER_LDFLAGS) -X 'main.trivyVersion=$(TRIVY_VERSION)'

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

OUTPUT_TYPE ?= type=docker
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)
GO_INSTALL := ./hack/go-install.sh

GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-v$(GOLANGCI_LINT_VERSION)

TEST_COUNT ?= 1
TIMEOUT ?= 1800s

$(GOLANGCI_LINT):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint $(GOLANGCI_LINT_BIN) v$(GOLANGCI_LINT_VERSION)

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
lint: $(GOLANGCI_LINT) ## Runs go linting.
	$(GOLANGCI_LINT) run -v

##@ Development

manifests: __controller-gen ## Generates k8s yaml for eraser deployment.
	@sed -e "s~ERASER_IMG~${ERASER_IMG}~g" \
		-e "s~COLLECTOR_IMG~${COLLECTOR_IMG}~g" \
		-e "s~SCANNER_IMG~${TRIVY_SCANNER_IMG}~g" \
		config/manager/kustomization.template.yaml > config/manager/kustomization.yaml
	docker run --rm -v $(shell pwd)/config:/config -w /config/manager \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} edit set image controller=${MANAGER_IMG}
	$(CONTROLLER_GEN) \
		crd \
		rbac:roleName=manager-role \
		webhook \
		paths="./..." \
		output:crd:artifacts:config=config/crd/bases
	rm -rf manifest_staging
	mkdir -p manifest_staging/deploy
	mkdir -p manifest_staging/charts/eraser
	docker run --rm -v $(shell pwd):/eraser \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/eraser/config/default -o /eraser/manifest_staging/deploy/eraser.yaml
	docker run --rm -v $(shell pwd):/eraser \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		--load_restrictor LoadRestrictionsNone /eraser/third_party/open-policy-agent/gatekeeper/helmify | go run third_party/open-policy-agent/gatekeeper/helmify/*.go

# Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method
# implementations. Also generate conversions between structs of different API versions.
generate: __conversion-gen __controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(CONVERSION_GEN) \
		--output-base=/eraser \
		--input-dirs=./api/... \
		--go-header-file=./hack/boilerplate.go.txt \
		--output-file-base=zz_generated.conversion

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: manifests generate fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

vulnerable-img:
	docker pull $(VULNERABLE_IMG)

non-vulnerable-img:
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${NON_VULNERABLE_IMG} \
		--target non-vulnerable .

e2e-test: vulnerable-img non-vulnerable-img
	for test in $(E2E_TESTS); do \
		CGO_ENABLED=0 \
			IMAGE=${ERASER_IMG} \
			MANAGER_IMAGE=${MANAGER_IMG} \
			COLLECTOR_IMAGE=${COLLECTOR_IMG} \
			SCANNER_IMAGE=${TRIVY_SCANNER_IMG} \
			VULNERABLE_IMAGE=${VULNERABLE_IMG} \
			NON_VULNERABLE_IMAGE=${NON_VULNERABLE_IMG} \
			NODE_VERSION=kindest/node:v${KUBERNETES_VERSION} \
			TEST_LOGDIR=${TEST_LOGDIR} \
			go test -count=$(TEST_COUNT) -timeout=$(TIMEOUT) $(TESTFLAGS) -tags=e2e -v $$test ; \
	done

##@ Build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager -ldflags "$(LDFLAGS)" main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build-manager: ## Build docker image with the manager.
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${MANAGER_IMG} \
		--target manager .

docker-build-trivy-scanner: ## Build docker image for trivy-scanner image.
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		--build-arg LDFLAGS="$(TRIVY_SCANNER_LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${TRIVY_SCANNER_IMG} \
		--target trivy-scanner .

docker-build-eraser: ## Build docker image for eraser image.
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		--build-arg LDFLAGS="$(ERASER_LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${ERASER_IMG} \
		--target eraser .

docker-build-collector:
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${COLLECTOR_IMG} \
		--target collector .

##@ Deployment

install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	docker run --rm -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/crd | kubectl apply -f -

uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	docker run --rm -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/crd | kubectl delete -f -

deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	docker run --rm -v $(shell pwd)/config:/config -w /config/manager \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} edit set image controller=${MANAGER_IMG}
	docker run --rm -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/default | kubectl apply --server-side -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	docker run --rm -v $(shell pwd)/config:/config \
		k8s.gcr.io/kustomize/kustomize:v${KUSTOMIZE_VERSION} build \
		/config/default | kubectl delete -f -

##@ Release

release-manifest: ## Generates manifests for a release.
	@sed -i -e 's/^VERSION := .*/VERSION := ${NEWVERSION}/' ./Makefile
	@sed -i'' -e 's@image: $(REPOSITORY):.*@image: $(REPOSITORY):'"$(NEWVERSION)"'@' ./config/manager/manager.yaml
	@sed -i "s/appVersion: .*/appVersion: ${NEWVERSION}/" ./third_party/open-policy-agent/gatekeeper/helmify/static/Chart.yaml
	@sed -i "s/version: .*/version: $$(echo ${NEWVERSION} | cut -c2-)/" ./third_party/open-policy-agent/gatekeeper/helmify/static/Chart.yaml
	@sed -i 's/Current release version: `.*`/Current release version: `'"${NEWVERSION}"'`/' ./third_party/open-policy-agent/gatekeeper/helmify/static/README.md
	export
	$(MAKE) manifests

promote-staging-manifest: ## Promotes the k8s deployment yaml files to release.
	@rm -rf deploy
	@cp -r manifest_staging/deploy .
	@rm -rf charts
	@cp -r manifest_staging/charts .

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: __tooling-image bin/setup-envtest

bin/setup-envtest:
	docker run --rm -v $(shell pwd)/bin:/go/bin -e GO111MODULE=on eraser-tooling go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

__controller-gen: __tooling-image
CONTROLLER_GEN=docker run --rm -v $(shell pwd):/eraser eraser-tooling controller-gen

__conversion-gen: __tooling-image
CONVERSION_GEN=docker run --rm -v $(shell pwd):/eraser eraser-tooling conversion-gen

__tooling-image:
	docker build . \
		-t eraser-tooling \
		-f build/tooling/Dockerfile

# Tags a new version for docs
.PHONY: version-docs
version-docs:
	docker run --rm \
		-v $(shell pwd)/docs:/docs \
		-w /docs \
		-u $(shell id -u):$(shell id -g) \
		node:${NODE_VERSION} \
		sh -c "yarn install --frozen-lockfile && yarn run docusaurus docs:version ${NEWVERSION}"
