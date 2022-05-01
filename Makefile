
# Image URL to use all building/pushing image targets
VERSION ?= dev
COMMIT ?= $(shell git rev-parse HEAD)
REPO ?= ghcr.io/solo-io/kdiag
IMG ?= $(REPO):$(VERSION)
LDFLAGS := "-X github.com/solo-io/kdiag/pkg/version.Version=$(VERSION) -X github.com/solo-io/kdiag/pkg/version.ImageRepo=$(REPO) -X github.com/solo-io/kdiag/pkg/version.Commit=$(COMMIT)"

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

.PHONY: 
scratch-shell/built/enter: scratch-shell/enter.c
	gcc $< -o $@

.PHONY: docker-build
docker-build:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --tag ${IMG} --build-arg=VERSION=$(VERSION) --build-arg=COMMIT=$(COMMIT) .

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

.PHONY: deploy-test-wokrloads
deploy-test-wokrloads:
	kubectl create deployment nginx --image=docker.io/library/nginx:1.19@sha256:2275af0f20d71b293916f1958f8497f987b8d8fd8113df54635f2a5915002bf1 || true
	kubectl expose deploy nginx --port 80 --target-port 80 || true
	kubectl create deployment curl --image=curlimages/curl@sha256:aa45e9d93122a3cfdf8d7de272e2798ea63733eeee6d06bd2ee4f2f8c4027d7c -- /bin/sh -c "while true; do curl --max-time 2 --head http://nginx; sleep 1; done"|| true

.PHONY: create-test-cluster
create-test-cluster:
	kind create cluster --image=docker.io/kindest/node:v1.23.0@sha256:49824ab1727c04e56a21a5d8372a402fcd32ea51ac96a2706a12af38934f81ac || true
	$(MAKE) deploy-test-wokrloads

.PHONY: reload-test-env
reload-test-env: docker-build
	kubectl delete pod -lapp=nginx || true
	kind load docker-image ${IMG}

.PHONY: create-test-env
create-test-env: create-test-cluster reload-test-env

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
