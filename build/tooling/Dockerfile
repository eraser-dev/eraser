FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195

# v0.16.4 is the minimum controller-gen release whose x/tools builds with Go 1.26.
RUN GOTOOLCHAIN=local GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
RUN GOTOOLCHAIN=local GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
