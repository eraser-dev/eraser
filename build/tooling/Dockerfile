FROM golang:1.23-bullseye@sha256:0c1625adb190cfc734a71d8e1350bc73ce29a0a8081849411a155fe5c9aaf302

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
