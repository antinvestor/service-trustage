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

# BAML requires CGO; build a dynamically-linked binary for glibc runtime.
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
     -ldflags="-s -w \
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
    && apt-get install -y --no-install-recommends ca-certificates libgcc-s1 libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

ENV HOME=/home/nonroot
ENV XDG_CACHE_HOME=/home/nonroot/.cache
ENV BAML_CACHE_DIR=/home/nonroot/.cache/baml

# Pre-download BAML shared library at build time so it works on read-only
# filesystems (migration jobs, hardened pods). Update BAML_VERSION when
# bumping the baml Go module in go.mod.
ARG BAML_VERSION=0.219.0
RUN mkdir -p /home/nonroot/.cache/baml \
    && apt-get update && apt-get install -y --no-install-recommends curl \
    && curl -fsSL "https://github.com/boundaryml/baml/releases/download/${BAML_VERSION}/libbaml_cffi-x86_64-unknown-linux-gnu.so" \
       -o /home/nonroot/.cache/baml/libbaml_cffi-x86_64-unknown-linux-gnu.so \
    && chmod 644 /home/nonroot/.cache/baml/libbaml_cffi-x86_64-unknown-linux-gnu.so \
    && chown -R 65532:65532 /home/nonroot \
    && apt-get purge -y curl && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*

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
