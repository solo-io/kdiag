# syntax=docker/dockerfile:1.3
# Build the manager binary

FROM --platform=${BUILDPLATFORM} docker.io/library/golang:1.18-bullseye as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY Makefile Makefile

# Build
RUN GOOS=linux GOARCH="${TARGETPLATFORM##linux/}" make build-manager

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM --platform=${TARGETPLATFORM} docker.io/library/ubuntu:20.04
ARG VERSION

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    lsb-release \
    gdb gcc libc6-dev \
    vim \
    iproute2 iptables \
    linux-tools-common linux-tools-generic \
    strace \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /

COPY --from=builder /workspace/manager /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]
