# syntax=mcr.microsoft.com/oss/moby/dockerfile:1.3.1
ARG BUILDERIMAGE="golang:1.18-bullseye"

ARG STATICBASEIMAGE="gcr.io/distroless/static:latest"
ARG STATICNONROOTBASEIMAGE="gcr.io/distroless/static:nonroot"

ARG TARGETOS
ARG TARGETARCH

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

FROM builder AS manager-build

RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o out/manager main.go

FROM builder AS eraser-build

RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags '-w -extldflags "-static"' -o out/eraser ./pkg/eraser

FROM --platform=$BUILDPLATFORM $STATICBASEIMAGE as eraser
COPY --from=eraser-build /workspace/out/eraser /
ENTRYPOINT ["/eraser"]

FROM builder AS collector-build

RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o out/collector ./pkg/collector

FROM --platform=$BUILDPLATFORM $STATICBASEIMAGE as collector
COPY --from=collector-build /workspace/out/collector /
ENTRYPOINT ["/collector"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=$BUILDPLATFORM $STATICNONROOTBASEIMAGE AS manager
WORKDIR /
COPY --from=manager-build /workspace/out/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
