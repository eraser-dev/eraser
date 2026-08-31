FROM golang:1.27.0-bookworm@sha256:ded31c68586d2e49e760acc2e65a884b23d032e9bbbed0ae0c55abd3fcaf4452

# v0.16.4 is the minimum controller-gen release whose x/tools builds with Go 1.26.
RUN GOTOOLCHAIN=local GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
RUN GOTOOLCHAIN=local GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@v0.29.0

RUN git config --global --add safe.directory "*"

WORKDIR /eraser
