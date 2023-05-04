FROM golang:1.20-bullseye@sha256:995b84ed1eb95c3aefa2787b00bad9129a6b0edc5363328bcde8b49d909fd4df

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0
RUN GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.25.4

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
