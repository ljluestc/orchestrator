#!/bin/bash
# Pre-commit test script - runs fast tests only

set -e

echo "Running pre-commit tests..."

# Run short tests only (skip long-running integration tests)
go test -short ./pkg/... ./cmd/... ./internal/... -timeout=30s

echo "✓ Pre-commit tests passed"
