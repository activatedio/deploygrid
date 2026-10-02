FROM golang:1.24.3 AS build-stage

ARG VERSION=dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

COPY pkg ./pkg
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X github.com/activatedio/deploygrid/pkg/collector.Version=${VERSION}" \
    -o /deploygrid ./cmd/main

# The same image runs the server and the collector. It is static and
# distroless: no shell, no package manager, no cloud CLIs. Clusters are
# observed by collectors running inside them (mode agent) or by the server
# through its own service account (mode local); kubeconfigs that need an
# external credential helper are not supported in this image.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build-stage /deploygrid /usr/bin/deploygrid

USER nonroot:nonroot

ENTRYPOINT ["/usr/bin/deploygrid"]
