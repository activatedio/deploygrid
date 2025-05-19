FROM golang:1.24.2 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

COPY pkg ./pkg
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux go build -o /deploygrid ./cmd/main

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /deploygrid /usr/bin/deploygrid

USER nonroot:nonroot

ENTRYPOINT ["/usr/bin/deploygrid"]
