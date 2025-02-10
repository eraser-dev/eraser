FROM golang:1.23-bullseye@sha256:80eb3147ef40b58d527d9c2e634b1a79a750aee09de6f973844db38b33f0550b

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
