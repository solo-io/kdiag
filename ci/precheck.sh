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

SHELL_IMG=$(make echo-shell-img)

# both docker files need to use the current shell image

grep $SHELL_IMG Dockerfile
grep $SHELL_IMG Dockerfile.relese

