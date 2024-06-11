FROM golang:1.22-bullseye@sha256:0742798b94c656e75ba5409583c2485f18b52de5c9b7cf12859c945bb7414718

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
