ARG TARGETOS=linux
ARG TARGETARCH=amd64

# ---------- Builder ----------
FROM golang:1.26 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app

ARG REPOSITORY
ARG VERSION=dev
ARG REVISION=none
ARG BUILDTIME

COPY go.mod go.sum ./
RUN go mod download

COPY ./apps/default ./apps/default
COPY ./baml_client ./baml_client
COPY ./dsl ./dsl
COPY ./connector ./connector
COPY ./pkg ./pkg

# BAML requires CGO; statically link with external linker for portable binary.
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
     -ldflags="-s -w -extldflags '-static' \
         -X github.com/pitabwire/frame/version.Repository=${REPOSITORY} \
         -X github.com/pitabwire/frame/version.Version=${VERSION} \
         -X github.com/pitabwire/frame/version.Commit=${REVISION} \
         -X github.com/pitabwire/frame/version.Date=${BUILDTIME}" \
     -o /app/binary ./apps/default/cmd/main.go

# ---------- Final ----------
# BAML downloads a native shared library built for glibc (x86_64-unknown-linux-gnu).
# Use a minimal Debian base with glibc and install libgcc runtime.
FROM debian:bookworm-slim

LABEL maintainer="Antinvestor <dev@antinvestor.com>"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libgcc-s1 \
    && rm -rf /var/lib/apt/lists/*

USER 65532:65532

EXPOSE 80

ARG REPOSITORY
ARG VERSION
ARG REVISION
ARG BUILDTIME
LABEL org.opencontainers.image.title="service-trustage"
LABEL org.opencontainers.image.version=$VERSION
LABEL org.opencontainers.image.revision=$REVISION
LABEL org.opencontainers.image.created=$BUILDTIME
LABEL org.opencontainers.image.source=$REPOSITORY

WORKDIR /

COPY --from=builder /app/binary /default
COPY --from=builder /app/apps/default/migrations /migrations

ENTRYPOINT ["/default"]
