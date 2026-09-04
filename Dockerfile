# syntax=docker/dockerfile:1.6

ARG GO_IMAGE="golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36"
# Use the upstream Trivy release image rather than rebuilding its binary with
# local dependency overrides.
ARG TRIVY_BINARY_IMG="ghcr.io/aquasecurity/trivy:0.72.0"
# HostProcess base image for the Windows workers. It is around 22 kB because a
# HostProcess container runs directly on the host and ships no OS of its own.
ARG HPC_BASE_IMG="mcr.microsoft.com/oss/kubernetes/windows-host-process-containers-base-image:v1.0.0"
ARG BUILDKIT_SBOM_SCAN_STAGE=builder,manager-build,collector-build,remover-build,trivy-scanner-build

FROM --platform=$TARGETPLATFORM $TRIVY_BINARY_IMG AS trivy-binary

# Build the manager binary
FROM --platform=$BUILDPLATFORM $GO_IMAGE AS builder
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ENV GOCACHE=/root/gocache
ENV CGO_ENABLED=0
RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .

ARG LDFLAGS
ARG TARGETOS
ARG TARGETARCH

FROM builder AS manager-build
RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ${LDFLAGS:+-ldflags "$LDFLAGS"} -o out/manager main.go

FROM builder AS collector-build
RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ${LDFLAGS:+-ldflags "$LDFLAGS"} -o out/collector ./pkg/collector

FROM builder AS remover-build
RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ${LDFLAGS:+-ldflags "$LDFLAGS"} -o out/remover ./pkg/remover

FROM builder AS trivy-scanner-build
RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ${LDFLAGS:+-ldflags "$LDFLAGS"} -o out/trivy-scanner ./pkg/scanners/trivy

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:nonroot AS manager
WORKDIR /
COPY --from=manager-build /workspace/out/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:latest as collector-linux
COPY --from=collector-build /workspace/out/collector /
ENTRYPOINT ["/collector"]

# A HostProcess container runs on the host, so "/collector" would resolve against
# the host filesystem. The image's own files are only reachable under the sandbox
# mount point, and cmd.exe comes from the host.
FROM --platform=windows/amd64 ${HPC_BASE_IMG} AS collector-windows
COPY --from=collector-build /workspace/out/collector /collector.exe
ENTRYPOINT ["cmd", "/c", "%CONTAINER_SANDBOX_MOUNT_POINT%\\collector.exe"]

FROM collector-${TARGETOS} AS collector

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:latest as remover-linux
COPY --from=remover-build /workspace/out/remover /
ENTRYPOINT ["/remover"]

FROM --platform=windows/amd64 ${HPC_BASE_IMG} AS remover-windows
COPY --from=remover-build /workspace/out/remover /remover.exe
ENTRYPOINT ["cmd", "/c", "%CONTAINER_SANDBOX_MOUNT_POINT%\\remover.exe"]

FROM remover-${TARGETOS} AS remover

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:latest AS trivy-scanner
COPY --from=trivy-scanner-build /workspace/out/trivy-scanner /
COPY --from=trivy-binary /usr/local/bin/trivy /
WORKDIR /var/lib/trivy
ENTRYPOINT ["/trivy-scanner"]

FROM gcr.io/distroless/static-debian12:nonroot AS non-vulnerable
COPY --from=builder /tmp /tmp
