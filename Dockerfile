# syntax=docker/dockerfile:1.6

ARG BUILDERIMAGE="golang:1.20-bullseye"
ARG TRIVY_BINARY_IMG="ghcr.io/aquasecurity/trivy:0.45.1"
ARG STATICBASEIMAGE="gcr.io/distroless/static:latest"
ARG STATICNONROOTBASEIMAGE="gcr.io/distroless/static:nonroot"
ARG BUILDKIT_SBOM_SCAN_STAGE=builder,manager-build,collector-build,remover-build,trivy-scanner-build

FROM --platform=$TARGETPLATFORM $TRIVY_BINARY_IMG AS trivy-binary

# Build the manager binary
FROM --platform=$BUILDPLATFORM $BUILDERIMAGE AS builder
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

FROM --platform=$TARGETPLATFORM $STATICNONROOTBASEIMAGE AS manager
WORKDIR /
COPY --from=manager-build /workspace/out/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]

FROM --platform=$TARGETPLATFORM $STATICBASEIMAGE as collector
COPY --from=collector-build /workspace/out/collector /
ENTRYPOINT ["/collector"]

FROM --platform=$TARGETPLATFORM $STATICBASEIMAGE as remover
COPY --from=remover-build /workspace/out/remover /
ENTRYPOINT ["/remover"]

FROM --platform=$TARGETPLATFORM $STATICBASEIMAGE as trivy-scanner
COPY --from=trivy-scanner-build /workspace/out/trivy-scanner /
COPY --from=trivy-binary /usr/local/bin/trivy /
WORKDIR /var/lib/trivy
ENTRYPOINT ["/trivy-scanner"]

FROM $STATICNONROOTBASEIMAGE as non-vulnerable
COPY --from=builder /tmp /tmp
