FROM golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36

# v0.16.4 is the minimum controller-gen release whose x/tools builds with Go 1.26.
RUN GOTOOLCHAIN=local GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
RUN GOTOOLCHAIN=local GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
