ARG TARGETOS
ARG TARGETARCH

# ---------- Builder ----------
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

ARG REPOSITORY
ARG VERSION=dev
ARG REVISION=none
ARG BUILDTIME

COPY go.mod go.sum ./
RUN go mod download

COPY ./apps/default ./apps/default
COPY ./dsl ./dsl
COPY ./connector ./connector
COPY ./gen ./gen
COPY ./pkg ./pkg
COPY ./proto ./proto

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w \
    -X github.com/pitabwire/frame/version.Repository=${REPOSITORY} \
    -X github.com/pitabwire/frame/version.Version=${VERSION} \
    -X github.com/pitabwire/frame/version.Commit=${REVISION} \
    -X github.com/pitabwire/frame/version.Date=${BUILDTIME}" \
    -o /app/binary ./apps/default/cmd/main.go

# ---------- Final ----------
FROM cgr.dev/chainguard/static:latest

LABEL maintainer="Antinvestor <dev@antinvestor.com>"

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
