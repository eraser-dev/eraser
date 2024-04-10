FROM golang:1.22-bullseye@sha256:d624e2db0a724a3f61a5234ff2433ae0ad708ada1ae4e960e1dd5a0b3994649d

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
