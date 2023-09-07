FROM golang:1.21-bullseye@sha256:873fce589a521deb54ec10f3deea6675135fed9ad91d62943a5c90d2fc059ba7

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
