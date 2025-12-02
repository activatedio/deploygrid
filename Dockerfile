FROM golang:1.24.3 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

COPY pkg ./pkg
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux go build -o /deploygrid ./cmd/main

# Deploy the application binary into a lean image
FROM debian:bookworm AS build-release-stage

RUN apt-get update && apt-get install -y awscli && rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

WORKDIR /

COPY --from=build-stage /deploygrid /usr/bin/deploygrid

ENTRYPOINT ["/usr/bin/deploygrid"]
