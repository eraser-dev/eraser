FROM golang:1.20-bullseye@sha256:b8b5510c6754ad783d8b7c3d1da3365c22cd02b25e2c42c94f8d353801c5cd0e

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
