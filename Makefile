VERSION := v1.4.0-beta.0

MANAGER_TAG ?= ${VERSION}
TRIVY_SCANNER_TAG ?= ${VERSION}
COLLECTOR_TAG ?= ${VERSION}
REMOVER_TAG ?= ${VERSION}

# Image URL to use all building/pushing image targets
TRIVY_SCANNER_REPO ?= ghcr.io/eraser-dev/eraser-trivy-scanner
TRIVY_SCANNER_IMG ?= ${TRIVY_SCANNER_REPO}:${TRIVY_SCANNER_TAG}
TRIVY_BINARY_REPO ?= ghcr.io/aquasecurity/trivy
TRIVY_BINARY_TAG ?= 0.48.3
TRIVY_BINARY_IMG ?= ${TRIVY_BINARY_REPO}:${TRIVY_BINARY_TAG}
MANAGER_REPO ?= ghcr.io/eraser-dev/eraser-manager
MANAGER_IMG ?= ${MANAGER_REPO}:${MANAGER_TAG}
REMOVER_REPO ?= ghcr.io/eraser-dev/remover
REMOVER_IMG ?= ${REMOVER_REPO}:${REMOVER_TAG}
COLLECTOR_REPO ?= ghcr.io/eraser-dev/collector
COLLECTOR_IMG ?= ${COLLECTOR_REPO}:${COLLECTOR_TAG}
VULNERABLE_IMG ?= docker.io/library/alpine:3.7.3
EOL_IMG ?= docker.io/library/alpine:3.1
BUSYBOX_BASE_IMG ?= busybox:1.36.0
NON_VULNERABLE_IMG ?= ghcr.io/eraser-dev/non-vulnerable:latest
E2E_TESTS ?= $(shell find ./test/e2e/tests/ -mindepth 1 -type d)
API_VERSIONS ?= ./api/v1alpha1,./api/v1,./api/v1alpha2,./api/v1alpha3

HELM_UPGRADE_TEST ?=
TEST_LOGDIR ?= $(PWD)/test_logs

REMOVER_TARBALL_PATH ?=
MANAGER_TARBALL_PATH ?=
COLLECTOR_TARBALL_PATH ?=
SCANNER_TARBALL_PATH ?=

KUSTOMIZE_VERSION ?= 3.8.9
KUBERNETES_VERSION ?= 1.25.3
NODE_VERSION ?= 20-bullseye-slim
ENVTEST_K8S_VERSION ?= 1.25
GOLANGCI_LINT_VERSION := 1.43.0

PLATFORM ?= linux

# build variables
LDFLAGS ?= $(shell build/version.sh "${VERSION}")
ERASER_LDFLAGS ?= -extldflags=-static $(LDFLAGS) -w
TRIVY_SCANNER_LDFLAGS ?= $(ERASER_LDFLAGS) -X 'main.trivyVersion=v$(TRIVY_BINARY_TAG)'

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

ifdef GENERATE_ATTESTATIONS
_ATTESTATIONS := --attest type=sbom --attest type=provenance,mode=max
endif

IDFLAGS=
ifeq (false,$(shell hack/rootless_docker.sh))
IDFLAGS=-u $(shell id -u):$(shell id -g)
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

#kustomize_

manifests: __manifest_kustomize __helm_kustomize __controller-gen ## Generates k8s yaml for eraser deployment.
	$(CONTROLLER_GEN) \
		crd \
		rbac:roleName=manager-role \
		webhook \
		paths="./..." \
		output:crd:artifacts:config=config/crd/bases
	rm -rf manifest_staging
	mkdir -p manifest_staging/deploy
	mkdir -p manifest_staging/charts/eraser
	$(MANIFEST_KUSTOMIZE) build /eraser/config/default -o /eraser/manifest_staging/deploy/eraser.yaml
	$(HELM_KUSTOMIZE) build \
		--load_restrictor LoadRestrictionsNone /eraser/third_party/open-policy-agent/gatekeeper/helmify | \
		go run third_party/open-policy-agent/gatekeeper/helmify/*.go

# Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method
# implementations. Also generate conversions between structs of different API versions.
generate: __conversion-gen __controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONVERSION_GEN) \
		--output-base=/eraser \
		--input-dirs=$(API_VERSIONS) \
		--go-header-file=./hack/boilerplate.go.txt \
		--output-file-base=zz_generated.conversion

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: manifests generate fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

busybox-img:
	docker build -t busybox-e2e-test:latest \
		-f test/e2e/test-data/Dockerfile.busybox \
		--build-arg IMG=$(BUSYBOX_BASE_IMG) test/e2e/test-data
BUSYBOX_IMG=busybox-e2e-test:latest

collector-dummy-img:
	docker build -t $(COLLECTOR_REPO):dummy \
		-f test/e2e/test-data/Dockerfile.dummyCollector \
		test/e2e/test-data
COLLECTOR_IMAGE_DUMMY=$(COLLECTOR_REPO):dummy

vulnerable-img:
	docker pull $(VULNERABLE_IMG)

eol-img:
	docker pull $(EOL_IMG)

non-vulnerable-img:
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${NON_VULNERABLE_IMG} \
		--target non-vulnerable .

custom-node-v$(KUBERNETES_VERSION):
	docker build -t custom-node:v$(KUBERNETES_VERSION) \
		-f test/e2e/test-data/Dockerfile.customNode \
		--build-arg KUBERNETES_VERSION=${KUBERNETES_VERSION} test/e2e/test-data
MODIFIED_NODE_IMAGE=custom-node:v$(KUBERNETES_VERSION)

e2e-test: vulnerable-img eol-img non-vulnerable-img busybox-img collector-dummy-img custom-node-v$(KUBERNETES_VERSION)
	for test in $(E2E_TESTS); do \
		CGO_ENABLED=0 \
            PROJECT_ABSOLUTE_PATH=$(CURDIR) \
            REMOVER_TARBALL_PATH=${REMOVER_TARBALL_PATH} \
            MANAGER_TARBALL_PATH=${MANAGER_TARBALL_PATH} \
            COLLECTOR_TARBALL_PATH=${COLLECTOR_TARBALL_PATH} \
            SCANNER_TARBALL_PATH=${SCANNER_TARBALL_PATH} \
			HELM_UPGRADE_TEST=${HELM_UPGRADE_TEST} \
			REMOVER_IMAGE=${REMOVER_IMG} \
			MANAGER_IMAGE=${MANAGER_IMG} \
			COLLECTOR_IMAGE=${COLLECTOR_IMG} \
			SCANNER_IMAGE=${TRIVY_SCANNER_IMG} \
			BUSYBOX_IMAGE=${BUSYBOX_IMG} \
			COLLECTOR_IMAGE_DUMMY=${COLLECTOR_IMAGE_DUMMY} \
			VULNERABLE_IMAGE=${VULNERABLE_IMG} \
			NON_VULNERABLE_IMAGE=${NON_VULNERABLE_IMG} \
			EOL_IMAGE=${EOL_IMG} \
			NODE_VERSION=kindest/node:v${KUBERNETES_VERSION} \
			MODIFIED_NODE_IMAGE=${MODIFIED_NODE_IMAGE} \
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
		$(_ATTESTATIONS) \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${MANAGER_IMG} \
		--target manager .

docker-build-trivy-scanner: ## Build docker image for trivy-scanner image.
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		$(_ATTESTATIONS) \
		--build-arg TRIVY_BINARY_IMG="$(TRIVY_BINARY_IMG)" \
		--build-arg LDFLAGS="$(TRIVY_SCANNER_LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${TRIVY_SCANNER_IMG} \
		--target trivy-scanner .

docker-build-remover: ## Build docker image for remover image.
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		$(_ATTESTATIONS) \
		--build-arg LDFLAGS="$(ERASER_LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${REMOVER_IMG} \
		--target remover .

docker-build-collector:
	docker buildx build \
		$(_CACHE_FROM) $(_CACHE_TO) \
		$(_ATTESTATIONS) \
		--build-arg LDFLAGS="$(LDFLAGS)" \
		--platform="$(PLATFORM)" \
		--output=$(OUTPUT_TYPE) \
		-t ${COLLECTOR_IMG} \
		--target collector .

##@ Deployment

install: __manifest_kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(MANIFEST_KUSTOMIZE) build /eraser/config/crd | kubectl apply -f -

uninstall: __manifest_kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(MANIFEST_KUSTOMIZE) build /eraser/config/crd | kubectl delete -f -

deploy: __manifest_kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(MANIFEST_KUSTOMIZE) build /eraser/config/default | kubectl apply -f -

undeploy: __manifest_kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(MANIFEST_KUSTOMIZE) build /eraser/config/default | kubectl delete -f -

##@ Release

release-manifest: ## Generates manifests for a release.
	@sed -i -e 's/^VERSION := .*/VERSION := ${NEWVERSION}/' ./Makefile
	@sed -i'' -e 's@image: $(REPOSITORY):.*@image: $(REPOSITORY):'"$(NEWVERSION)"'@' ./config/manager/manager.yaml
	@sed -i "s/appVersion: .*/appVersion: ${NEWVERSION}/" ./third_party/open-policy-agent/gatekeeper/helmify/static/Chart.yaml
	@sed -i "s/version: .*/version: $$(echo ${NEWVERSION} | cut -c2-)/" ./third_party/open-policy-agent/gatekeeper/helmify/static/Chart.yaml
	@sed -Ei 's/(tag:\s*).*/\1"$(NEWVERSION)"/' ./third_party/open-policy-agent/gatekeeper/helmify/static/values.yaml
	@sed -i 's/Current release version: `.*`/Current release version: `'"${NEWVERSION}"'`/' ./third_party/open-policy-agent/gatekeeper/helmify/static/README.md
	@sed -i 's/https:\/\/raw\.githubusercontent\.com\/eraser-dev\/eraser\/master\/deploy\/eraser\.yaml.*/https:\/\/raw\.githubusercontent\.com\/eraser-dev\/eraser\/${NEWVERSION}\/deploy\/eraser\.yaml/' ./docs/docs/installation.md
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

__manifest_kustomize: __kustomize-manifest-image
MANIFEST_KUSTOMIZE=docker run --rm -v $(shell pwd)/manifest_staging:/eraser/manifest_staging manifest-kustomize

__helm_kustomize: __kustomize-helm-image
HELM_KUSTOMIZE=docker run --rm -v $(shell pwd)/manifest_staging:/eraser/manifest_staging -v $(shell pwd)/third_party:/eraser/third_party helm-kustomize

__tooling-image:
	docker build . \
		-t eraser-tooling \
		-f build/tooling/Dockerfile

__kustomize-helm-image:
	docker build . \
		-t helm-kustomize \
		--build-arg KUSTOMIZE_VERSION=${KUSTOMIZE_VERSION} \
		-f build/tooling/Dockerfile.helm

__kustomize-manifest-image:
	docker build . \
		-t manifest-kustomize \
		--build-arg KUSTOMIZE_VERSION=${KUSTOMIZE_VERSION} \
		--build-arg TRIVY_SCANNER_REPO=${TRIVY_SCANNER_REPO} \
		--build-arg MANAGER_REPO=${MANAGER_REPO} \
		--build-arg REMOVER_REPO=${REMOVER_REPO} \
		--build-arg COLLECTOR_REPO=${COLLECTOR_REPO} \
		--build-arg MANAGER_TAG=${MANAGER_TAG} \
		--build-arg TRIVY_SCANNER_TAG=${TRIVY_SCANNER_TAG} \
		--build-arg COLLECTOR_TAG=${COLLECTOR_TAG} \
		--build-arg REMOVER_TAG=${REMOVER_TAG} \
		-f build/tooling/Dockerfile.manifest

# Tags a new version for docs
.PHONY: version-docs
version-docs:
	docker run --rm \
		-v $(shell pwd)/docs:/docs \
		-w /docs \
		$(IDFLAGS) \
		node:${NODE_VERSION} \
		sh -c "yarn install --frozen-lockfile && yarn run docusaurus docs:version ${NEWVERSION}"
	@sed -i 's/https:\/\/raw\.githubusercontent\.com\/eraser-dev\/eraser\/main\/deploy\/eraser\.yaml.*/https:\/\/raw\.githubusercontent\.com\/eraser-dev\/eraser\/${TAG}\/deploy\/eraser\.yaml/' ./docs/versioned_docs/version-${NEWVERSION}/installation.md

.PHONY: patch-version-docs
patch-version-docs:
	@sed -i 's/https:\/\/raw\.githubusercontent\.com\/eraser-dev\/eraser\/${OLDVERSION}\/deploy\/eraser\.yaml.*/https:\/\/raw\.githubusercontent\.com\/eraser-dev\/eraser\/${TAG}\/deploy\/eraser\.yaml/' ./docs/versioned_docs/version-${NEWVERSION}/installation.md
