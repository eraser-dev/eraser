FROM golang:1.18-bullseye

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0

WORKDIR /build
COPY build/tooling/go.mod build/tooling/go.sum ./
COPY build/tooling/tools.go ./
RUN GO111MODULE=on go mod vendor

WORKDIR /eraser
