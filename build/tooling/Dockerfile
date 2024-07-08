FROM golang:1.22-bullseye@sha256:11ff2c5de8044d8d749cc684f6f3cdfbc9b2755960a62d4948b3483dc090e7aa

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
