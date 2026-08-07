FROM golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651

# v0.16.4 is the minimum controller-gen release whose x/tools builds with Go 1.26.
RUN GOTOOLCHAIN=local GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
RUN GOTOOLCHAIN=local GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
