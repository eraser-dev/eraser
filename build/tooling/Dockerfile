FROM golang:1.21-bullseye@sha256:d9203b088f9e50cc60eccfc0129d9bd5109b236a5e5cda1a4527aef834fd85ef

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
