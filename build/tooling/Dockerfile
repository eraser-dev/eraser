FROM golang:1.23-bullseye@sha256:bc1b90c2a8eb0ffb62325e02a85d51031ad3afae15b3df4b6a48b7929b00badb

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
