FROM golang:1.22-bullseye@sha256:72885e2245d6bcc63af0538043c63454878a22733587af87a4cfb12268d03baf

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
