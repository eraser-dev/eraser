FROM golang:1.23-bullseye@sha256:6d4cbf0b3900afa3e4460ca995b6c351370ce8d2d44b7a964dc521ab640e1a88

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
