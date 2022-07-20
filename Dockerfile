FROM --platform=${BUILDPLATFORM} docker.io/library/golang:1.18-bullseye as builder
# Build the manager binary

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG SHELL_IMG

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

FROM --platform=${TARGETPLATFORM} ghcr.io/solo-io/kdiag-shell:020030


WORKDIR /

COPY --from=builder /workspace/manager /usr/local/bin/manager
ENTRYPOINT ["/usr/local/bin/manager"]
