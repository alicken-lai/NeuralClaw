#!/usr/bin/env bash

# Dev script to build and run local
set -e

echo "Building zclaw..."
go build -o bin/zclaw ./cmd/zclaw

echo "Run 'bin/zclaw --help' to test."
