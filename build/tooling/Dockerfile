FROM golang:1.21-bullseye@sha256:1ad88350bb991ac82b71124815a504806e5817d171f5b08cc63d9c30bd8e5970

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
