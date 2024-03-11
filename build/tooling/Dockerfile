FROM golang:1.22-bullseye@sha256:9f880bd713b1e3d22d50d968e6369c6d7f750112f8e52c52b8a75756eb448d81

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
