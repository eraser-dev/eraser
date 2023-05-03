FROM golang:1.20-bullseye@sha256:d282e704848c81e313b17c68e27edd44202937db196398915077830557f82588

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
