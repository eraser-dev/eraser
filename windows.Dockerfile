# syntax=mcr.microsoft.com/oss/moby/dockerfile:1.3.1
ARG BUILDERIMAGE="golang:1.18-bullseye"

ARG TARGETOS=windows
ARG TARGETARCH=amd64
ARG OSVERSION

# Build the manager binary
FROM --platform=linux/amd64 $BUILDERIMAGE AS builder
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

FROM builder AS eraser-build

RUN \
    --mount=type=cache,target=${GOCACHE} \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags '-w -extldflags "-static"' -o out/eraser.exe ./pkg/eraser

FROM --platform=linux/amd64 gcr.io/k8s-staging-e2e-test-images/windows-servercore-cache:1.0-linux-amd64-${OSVERSION} as core

FROM mcr.microsoft.com/windows/nanoserver:${OSVERSION}
COPY --from=eraser-build /workspace/out/eraser.exe /eraser.exe
COPY --from=core /Windows/System32/netapi32.dll /Windows/System32/netapi32.dll

ENTRYPOINT ["/eraser.exe"]