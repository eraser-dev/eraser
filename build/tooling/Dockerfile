FROM golang:1.18-bullseye
ARG KUBERNETES_CODE_GENERATOR_VERSION

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0

WORKDIR /build
COPY build/tooling/go.mod ./
COPY build/tooling/tools.go ./
RUN GO111MODULE=on go get k8s.io/code-generator@${KUBERNETES_CODE_GENERATOR_VERSION}
RUN GO111MODULE=on go mod vendor

WORKDIR /eraser
