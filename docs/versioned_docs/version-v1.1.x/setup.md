---
title: Setup
---

# Development Setup

This document describes the steps to get started with development.
You can either utilize [Codespaces](https://docs.github.com/en/codespaces/overview) or setup a local environment.

## Local Setup

### Prerequisites:

- [go](https://go.dev/) with version 1.17 or later.
- [docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/)
- `make`

### Get things running

- Get dependencies with `go get`

- This project uses `make`. You can utilize `make help` to see available targets. For local deployment make targets help to build, test and deploy.

### Making changes

Please refer to [Development Reference](#development-reference) for more details on the specific commands.

To test your changes on a cluster:

```bash
# generate necessary api files (optional - only needed if changes to api folder).
make generate

# build applicable images
make docker-build-manager MANAGER_IMG=eraser-manager:dev
make docker-build-remover REMOVER_IMG=remover:dev
make docker-build-collector COLLECTOR_IMG=collector:dev
make docker-build-trivy-scanner TRIVY_SCANNER_IMG=eraser-trivy-scanner:dev

# make sure updated image is present on cluster (e.g., see kind example below)
kind load docker-image \
        eraser-manager:dev \
        eraser-trivy-scanner:dev \
        remover:dev \
        collector:dev

make manifests
make deploy

# to remove the deployment
make undeploy
```

To test your changes to manager locally:

```bash
make run
```

Example Output:

```
you@local:~/eraser$ make run
docker build . \
        -t eraser-tooling \
        -f build/tooling/Dockerfile
[+] Building 7.8s (8/8) FINISHED
 => => naming to docker.io/library/eraser-tooling                           0.0s
docker run -v /home/eraser/config:/config -w /config/manager \
        registry.k8s.io/kustomize/kustomize:v3.8.9 edit set image controller=eraser-manager:dev
docker run -v /home/eraser:/eraser eraser-tooling controller-gen \
        crd \
        rbac:roleName=manager-role \
        webhook \
        paths="./..." \
        output:crd:artifacts:config=config/crd/bases
rm -rf manifest_staging
mkdir -p manifest_staging/deploy
docker run --rm -v /home/eraser:/eraser \
        registry.k8s.io/kustomize/kustomize:v3.8.9 build \
        /eraser/config/default -o /eraser/manifest_staging/deploy/eraser.yaml
docker run -v /home/eraser:/eraser eraser-tooling controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go run ./main.go
{"level":"info","ts":1652985685.1663408,"logger":"controller-runtime.metrics","msg":"Metrics server is starting to listen","addr":":8080"}
...
```

## Development Reference

Eraser is using tooling from [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder). For Eraser this tooling is containerized into the `eraser-tooling` image. The `make` targets can use this tooling and build the image when necessary.

You can override the default configuration using environment variables. Below you can find a reference of targets and configuration options.

### Common Configuration

| Environment Variable | Description                                                                                   |
| -------------------- | --------------------------------------------------------------------------------------------- |
| VERSION              | Specifies the version (i.e., the image tag) of eraser to be used.                             |
| MANAGER_IMG          | Defines the image url for the Eraser manager. Used for tagging, pulling and pushing the image |
| REMOVER_IMG           | Defines the image url for the Eraser. Used for tagging, pulling and pushing the image         |
| COLLECTOR_IMG        | Defines the image url for the Collector. Used for tagging, pulling and pushing the image      |

### Linting

- `make lint`

Lints the go code.

| Environment Variable | Description                                             |
| -------------------- | ------------------------------------------------------- |
| GOLANGCI_LINT        | Specifies the go linting binary to be used for linting. |

### Development

- `make generate`

Generates necessary files for the k8s api stored under `api/v1alpha1/zz_generated.deepcopy.go`. See the [kubebuilder docs](https://book.kubebuilder.io/cronjob-tutorial/other-api-files.html) for details.

- `make manifests`

Generates the eraser deployment yaml files under `manifest_staging/deploy`.

Configuration Options:

| Environment Variable | Description                                        |
| -------------------- | -------------------------------------------------- |
| REMOVER_IMG           | Defines the image url for the Eraser.              |
| MANAGER_IMG          | Defines the image url for the Eraser manager.      |
| KUSTOMIZE_VERSION    | Define Kustomize version for generating manifests. |

- `make test`

Runs the unit tests for the eraser project.

Configuration Options:

| Environment Variable | Description                                                 |
| -------------------- | ----------------------------------------------------------- |
| ENVTEST              | Specifies the envtest setup binary.                         |
| ENVTEST_K8S_VERSION  | Specifies the Kubernetes version for envtest setup command. |

- `make e2e-test`

Runs e2e tests on a cluster.

Configuration Options:

| Environment Variable | Description                                                                                                   |
| -------------------- | ------------------------------------------------------------------------------------------------------------- |
| REMOVER_IMG           | Eraser image to be used for e2e test.                                                                         |
| MANAGER_IMG          | Eraser manager image to be used for e2e test.                                                                 |
| KUBERNETES_VERSION   | Kubernetes version for e2e test.                                                                              |
| TEST_COUNT           | Sets repetition for test. Please refer to [go docs](https://pkg.go.dev/cmd/go#hdr-Testing_flags) for details. |
| TIMEOUT              | Sets timeout for test. Please refer to [go docs](https://pkg.go.dev/cmd/go#hdr-Testing_flags) for details.    |
| TESTFLAGS            | Sets additional test flags                                                                                    |

### Build

- `make build`

Builds the eraser manager binaries.

- `make run`

Runs the eraser manager on your local machine.

- `make docker-build-manager`

Builds the docker image for the eraser manager.

Configuration Options:

| Environment Variable | Description                                                                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| CACHE_FROM           | Sets the target of the buildx --cache-from flag [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-from). |
| CACHE_TO             | Sets the target of the buildx --cache-to flag [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-to).     |
| PLATFORM             | Sets the target platform for buildx [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#platform).               |
| OUTPUT_TYPE          | Sets the output for buildx [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#output).                          |
| MANAGER_IMG          | Specifies the target repository, image name and tag for building image.                                                                                |

- `make docker-push-manager`

Builds the docker image for the eraser manager.

Configuration Options:

| Environment Variable | Description                                                             |
| -------------------- | ----------------------------------------------------------------------- |
| MANAGER_IMG          | Specifies the target repository, image name and tag for building image. |

- `make docker-build-remover`

Builds the docker image for eraser remover.

Configuration Options:

| Environment Variable | Description                                                                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| CACHE_FROM           | Sets the target of the buildx --cache-from flag [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-from). |
| CACHE_TO             | Sets the target of the buildx --cache-to flag [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-to).     |
| PLATFORM             | Sets the target platform for buildx [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#platform).               |
| OUTPUT_TYPE          | Sets the output for buildx [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#output).                          |
| REMOVER_IMG           | Specifies the target repository, image name and tag for building image.                                                                                |

- `make docker-push-remover`

Builds the docker image for the eraser remover.

Configuration Options:

| Environment Variable | Description                                                             |
| -------------------- | ----------------------------------------------------------------------- |
| REMOVER_IMG           | Specifies the target repository, image name and tag for building image. |

- `make docker-build-collector`

Builds the docker image for the eraser collector.

Configuration Options:

| Environment Variable | Description                                                                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| CACHE_FROM           | Sets the target of the buildx --cache-from flag [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-from). |
| CACHE_TO             | Sets the target of the buildx --cache-to flag [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-to).     |
| PLATFORM             | Sets the target platform for buildx [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#platform).               |
| OUTPUT_TYPE          | Sets the output for buildx [see buildx reference](https://docs.docker.com/engine/reference/commandline/buildx_build/#output).                          |
| COLLECTOR_IMG        | Specifies the target repository, image name and tag for building image.                                                                                |

- `make docker-push-collector`

Builds the docker image for the eraser collector.

Configuration Options:

| Environment Variable | Description                                                             |
| -------------------- | ----------------------------------------------------------------------- |
| COLLECTOR_IMG        | Specifies the target repository, image name and tag for building image. |

### Deployment

- `make install`

Install CRDs into the K8s cluster specified in ~/.kube/config.

Configuration Options:

| Environment Variable | Description                                                      |
| -------------------- | ---------------------------------------------------------------- |
| KUSTOMIZE_VERSION    | Kustomize version used to generate k8s resources for deployment. |

- `make uninstall`

Uninstall CRDs from the K8s cluster specified in ~/.kube/config.

Configuration Options:

| Environment Variable | Description                                                      |
| -------------------- | ---------------------------------------------------------------- |
| KUSTOMIZE_VERSION    | Kustomize version used to generate k8s resources for deployment. |

- `make deploy`

Deploys eraser to the cluster specified in ~/.kube/config.

Configuration Options:

| Environment Variable | Description                                                          |
| -------------------- | -------------------------------------------------------------------- |
| KUSTOMIZE_VERSION    | Kustomize version used to generate k8s resources for deployment.     |
| MANAGER_IMG          | Specifies the eraser manager image version to be used for deployment |

- `make undeploy`

Undeploy controller from the K8s cluster specified in ~/.kube/config.

Configuration Options:

| Environment Variable | Description                                                               |
| -------------------- | ------------------------------------------------------------------------- |
| KUSTOMIZE_VERSION    | Kustomize version used to generate k8s resources that need to be removed. |

### Release

- `make release-manifest`

Generates k8s manifests files for a release.

Configuration Options:

| Environment Variable | Description                          |
| -------------------- | ------------------------------------ |
| NEWVERSION           | Sets the new version in the Makefile |

- `make promote-staging-manifest`

Promotes the k8s deployment yaml files to release.
