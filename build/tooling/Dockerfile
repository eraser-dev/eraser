FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea

# v0.16.4 is the minimum controller-gen release whose x/tools builds with Go 1.26.
RUN GOTOOLCHAIN=local GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
RUN GOTOOLCHAIN=local GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
