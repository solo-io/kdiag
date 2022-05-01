FROM --platform=${BUILDPLATFORM} docker.io/library/golang:1.18-bullseye as builder
# Build the manager binary

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

FROM --platform=${TARGETPLATFORM} docker.io/library/ubuntu:22.04

# Install dependencies
# note: we can't use linux-headers-$(uname -r), as we don't know the host kernel will match.
# we just need some version of the headers so we can build busybox.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    lsb-release \
    gdb gcc libc6-dev \
    vim \
    iproute2 iptables nftables \
    strace make \
    linux-libc-dev linux-headers-5.15.0-27-generic \
    && rm -rf /var/lib/apt/lists/*

# RUN ln -s /usr/include/asm-generic /usr/include/asm
RUN ln -s /usr/include/*-linux-gnu/asm/ /usr/include/asm

COPY scratch-shell/.config scratch-shell/build.sh scratch-shell/enter.c /scratch-shell/
RUN cd /scratch-shell && ./build.sh && \
    cp ./built/ash /usr/local/bin && \
    gcc ./enter.c -o /usr/local/bin/enter

WORKDIR /

COPY --from=builder /workspace/manager /usr/local/bin/manager
ENTRYPOINT ["/usr/local/bin/manager"]
