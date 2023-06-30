FROM golang:1.20-bullseye@sha256:4c94dfe9567ee66a0ebc35a28f9f939e9ca3a71cb2ab942359887c584da1b28e

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
