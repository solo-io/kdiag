#!/bin/bash

set -e
set -x

# check that code generation is up to date
make generate
go mod tidy
go vet
if [[ $(git status --porcelain | wc -l) -ne 0 ]]; then
    echo "Generating code produced a non-empty diff"
    echo "Try running 'make generate && go mod tidy' then re-pushing."
    git status --porcelain
    git diff | cat
    exit 1;
fi

# make sure release and dev dockerfile are in sync
diff <(grep "Install dependencies" -A1000 Dockerfile | head -n -3) <(grep "Install dependencies" -A1000 Dockerfile.release | head -n -3)

# run tests
kind create cluster --image=docker.io/kindest/node:v1.23.0@sha256:49824ab1727c04e56a21a5d8372a402fcd32ea51ac96a2706a12af38934f81ac
make create-test-env
go test ./...
