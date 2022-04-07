#!/bin/bash

set -x

# check that code generation is up to date
make generate
go mod tidy
go vet
if [[ $(git status --porcelain | wc -l) -ne 0 ]]; then
    echo "Generating code produced a non-empty diff"
    echo "Try running 'make generate && go mod tidy && go vet' then re-pushing."
    git status --porcelain
    git diff | cat
    exit 1;
fi

set -e

# make sure release and dev dockerfile are in sync
diff <(grep "Install dependencies" -A1000 Dockerfile | head -n -3) <(grep "Install dependencies" -A1000 Dockerfile.release | head -n -3)
