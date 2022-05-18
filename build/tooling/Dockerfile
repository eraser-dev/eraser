FROM golang:1.18

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0

RUN mkdir /eraser
WORKDIR /eraser
