FROM golang:1.23-bullseye@sha256:462521f1b7cbf410002a8cc4d91bc897f35cd430854d7240596282f9441fe4a7

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
