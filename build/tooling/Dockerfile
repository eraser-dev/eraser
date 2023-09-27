FROM golang:1.21-bullseye@sha256:18f57bdd1d188ade1226471393b5c3ced4639c6c772fd94853ea7abd9eb158f7

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
