FROM golang:1.22-bullseye@sha256:583d5af8289d30de50aa0dcf4985d8b8746e52622becd6e1a62cfe191d5275a5

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
