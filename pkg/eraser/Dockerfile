FROM golang:1.17 AS build

WORKDIR /src

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .

RUN \
    --mount=type=cache,target=/root/gocache \
    cd pkg/eraser && GOCACHE=/root/gocache CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -extldflags "-static"' -o /out/removal .

RUN chmod +x /out/removal

FROM scratch as bin

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /out/removal /

ENTRYPOINT ["/removal"]
