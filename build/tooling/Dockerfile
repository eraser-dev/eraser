FROM golang:1.22-bullseye@sha256:dcff0d950cb4648fec14ee51baa76bf27db3bb1e70a49f75421a8828db7b9910

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
