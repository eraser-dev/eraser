FROM golang:1.23-bullseye@sha256:ebaf58d74f200c27dce0181203a1347e41b5c159c9aa0517a95bec594aabbde4

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
