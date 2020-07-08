#!/bin/bash

# test_all.sh

unset ok
[ -z "$USE_DEFAULT_CONFIG" ] || ok=1
[ -f config.test.toml ] && ok=1
if [ "$ok" != "1" ]; then
  echo 'This will fail unless you run "make testdeps" first or set USE_DEFAULT_CONFIG=1' 1>&2
  exit 2
fi

if [ -z ${GO_ENV} ]; then
    export GO_ENV="test"
fi

export GO111MODULE=on

# Run the unit tests with the race detector and code coverage enabled
set -xeuo pipefail
go test -race -coverprofile cover.out -covermode atomic ./...
