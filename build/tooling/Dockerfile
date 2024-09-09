FROM golang:1.23-bullseye@sha256:45b43371f21ec51276118e6806a22cbb0bca087ddd54c491fdc7149be01035d5

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
