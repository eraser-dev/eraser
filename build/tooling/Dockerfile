FROM golang:1.22-bullseye@sha256:37f09f0c199a07c2e72ae2cfd758681fae0681240c71d6fad42d9d090c437c38

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
