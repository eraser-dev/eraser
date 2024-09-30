FROM golang:1.23-bullseye@sha256:1a26d5ad2e9bdbe9206f1db3035dbf90ad2b4ad09ccbbbcf5ec0c4e56bbe77d1

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
