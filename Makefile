
# Image URL to use all building/pushing image targets
VERSION ?= dev
COMMIT ?= $(shell git rev-parse HEAD)
IMG ?= ghcr.io/solo-io/kdiag:$(VERSION)
LDFLAGS := "-X github.com/solo-io/kdiag/pkg/version.Version=$(VERSION) -X github.com/solo-io/kdiag/pkg/version.Commit=$(COMMIT)"

.PHONY: all
all: docker-build

.PHONY: ginkgo-test
test: ginkgo generate fmt vet
	$(GINKGO) -r --coverprofile cover.out --race

.PHONY: generate
generate: protoc-gen-go
	PATH=$(shell pwd)/bin:$$PATH go generate ./...
	go run pkg/cmd/scripts/docs.go

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: docker-build
docker-build:
	DOCKER_BUILDKIT=1 docker build --tag ${IMG} --build-arg=VERSION=$(VERSION) --build-arg=COMMIT=$(COMMIT) .

.PHONY: docker-build-push
docker-build-push:
	DOCKER_BUILDKIT=1 docker buildx build --platform linux/amd64,linux/arm64 --tag ${IMG} --build-arg=VERSION=$(VERSION) --build-arg=COMMIT=$(COMMIT) .

build-manager:
	CGO_ENABLED=0 go build -a -o manager -ldflags=$(LDFLAGS) cmd/srv/srv.go

GINKGO = $(shell pwd)/bin/ginkgo
.PHONY: ginkgo
ginkgo: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.1.0)

PROTOC_GEN_GO = $(shell pwd)/bin/protoc-gen-go
.PHONY: protoc-gen-go
protoc-gen-go: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(PROTOC_GEN_GO),google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1)

create-test-env: docker-build
	kind create cluster || true
	kubectl delete pod -lapp=nginx || true
	kubectl create deployment nginx --image=docker.io/library/nginx:1.19 || true
	kubectl expose deploy nginx --port 80 --target-port 80 || true
	kind load docker-image ${IMG}
	kubectl create deployment curl --image=${IMG} -- bash -c "while true; do curl --max-time 2 --head http://nginx; sleep 1; done"|| true

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
