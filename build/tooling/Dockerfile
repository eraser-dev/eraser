FROM golang:1.22-bullseye@sha256:0c099cf97c6162ceab8c9494c4ce8c2a19a73b852f334ec1876a42d05eb5f90b

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
