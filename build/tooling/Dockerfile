FROM golang:1.24-bullseye@sha256:dfd72198d14bc22f270c9e000c304a2ffd19f5a5f693fad82643311afdc6b568

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
