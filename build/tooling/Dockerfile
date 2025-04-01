FROM golang:1.24-bullseye@sha256:cf29cafe674ad5e637311148fb7933f67c4e8cafa79ce066aad0e7aa708fc287

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
