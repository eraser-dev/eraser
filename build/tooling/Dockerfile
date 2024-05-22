FROM golang:1.22-bullseye@sha256:78b171fe51f25b8c3197710f281dacae94759a254e9a486576005f9dadba9e7d

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
