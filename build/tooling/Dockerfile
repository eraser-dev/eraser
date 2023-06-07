FROM golang:1.20-bullseye@sha256:419bc8954c0e08c539830c8669ccd116a063303481c748fabd09d8fd6d4e2c5f

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
