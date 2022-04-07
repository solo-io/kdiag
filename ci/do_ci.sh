#!/bin/bash
# run tests

set -e
set -x

make create-test-env

go test ./...
