# Comprehensive Testing Guide for Orchestrator Platform

**Version:** 1.0
**Last Updated:** 2025-10-18
**Target Coverage:** 100%

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Test Organization](#test-organization)
4. [Running Tests](#running-tests)
5. [Coverage Analysis](#coverage-analysis)
6. [Writing Tests](#writing-tests)
7. [CI/CD Integration](#cicd-integration)
8. [Troubleshooting](#troubleshooting)

---

## Overview

The Orchestrator platform uses a comprehensive testing strategy to ensure code quality and reliability:

- **Unit Tests**: Test individual functions and methods in isolation
- **Integration Tests**: Test component interactions and end-to-end workflows
- **Benchmark Tests**: Measure performance characteristics
- **Race Detection**: Identify concurrency issues

### Current Status

| Package | Coverage | Status |
|---------|----------|--------|
| internal/storage | 96.1% | ✅ Excellent |
| pkg/app | 93.5% | ✅ Excellent |
| pkg/mesos/agent | 91.1% | ✅ Excellent |
| pkg/topology | 86.2% | ✅ Good |
| pkg/isolation | 85.7% | ✅ Good |
| pkg/probe | 84.4% | ⚠️  Needs Improvement |
| pkg/security | 77.5% | ⚠️  Needs Improvement |
| pkg/containerizer | 72.2% | ⚠️  Needs Improvement |
| pkg/migration | 59.0% | ❌ Critical |
| cmd/app | 53.7% | ❌ Critical |
| cmd/probe | 0.0% | ❌ Critical |

**Overall Coverage:** 85.2%
**Target Coverage:** 100%

---

## Quick Start

### Prerequisites

```bash
# Install Go 1.23+
go version

# Install testing tools
go install golang.org/x/tools/cmd/cover@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install pre-commit hooks
pip install pre-commit
pre-commit install
```

### Run All Tests

```bash
# Quick test (unit tests only, no race detection)
go test ./... -short

# Full test suite with coverage
./scripts/test_orchestrator.sh

# With race detection
go test -race ./...

# Integration tests
go test -tags=integration ./...

# Specific package
go test -v -cover ./pkg/probe/...
```

---

## Test Organization

### Directory Structure

```
orchestrator/
├── cmd/
│   ├── app/
│   │   ├── main.go
│   │   ├── main_test.go           # CLI integration tests
│   │   └── server_test.go          # Server lifecycle tests
│   └── probe/
│       ├── main.go
│       └── main_test.go            # Probe CLI tests
├── pkg/
│   ├── app/
│   │   ├── *.go                    # Source files
│   │   └── *_test.go               # Unit tests
│   ├── probe/
│   │   ├── *.go
│   │   ├── *_test.go               # Standard tests
│   │   ├── integration_test.go     # Integration tests
│   │   └── benchmark_test.go       # Performance tests
│   └── .../
├── internal/
│   └── storage/
│       ├── *.go
│       └── *_test.go
├── integration_test.go              # System-wide integration tests
└── scripts/
    └── test_orchestrator.sh         # Comprehensive test runner
```

### Test File Naming Conventions

- **Unit tests**: `*_test.go` - In same package as source
- **Integration tests**: `integration_test.go` or tagged with `// +build integration`
- **Benchmark tests**: `benchmark_test.go` or `*_benchmark_test.go`
- **Table-driven tests**: Use descriptive test names

---

## Running Tests

### Basic Test Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests with race detection
go test -race ./...

# Run tests with timeout
go test -timeout=10m ./...

# Run specific test
go test -run TestProbeCollect ./pkg/probe/...

# Run tests matching pattern
go test -run "TestProbe.*" ./...
```

### Advanced Testing

```bash
# Comprehensive test suite (recommended)
./scripts/test_orchestrator.sh

# With custom coverage target
./scripts/test_orchestrator.sh --coverage-target 95.0

# Skip integration tests
./scripts/test_orchestrator.sh --skip-integration

# Run benchmarks
./scripts/test_orchestrator.sh --run-benchmarks

# Generate JSON coverage report
./scripts/test_orchestrator.sh
cat coverage/coverage.json
```

### Integration Tests

```bash
# Run integration tests only
go test -tags=integration -v ./...

# Run system integration test
go test -tags=integration -v ./integration_test.go

# Skip integration tests
go test -short ./...
```

### Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkProbeCollect ./pkg/probe/...

# Run with custom benchmark time
go test -bench=. -benchtime=10s ./...

# Compare benchmarks
go test -bench=. ./pkg/probe/... > old.txt
# Make changes
go test -bench=. ./pkg/probe/... > new.txt
benchcmp old.txt new.txt
```

---

## Coverage Analysis

### Generating Coverage Reports

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage by function
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Package-Level Coverage

```bash
# Test specific package with coverage
go test -coverprofile=pkg_coverage.out ./pkg/probe/...

# Get coverage percentage
go tool cover -func=pkg_coverage.out | grep total

# Find uncovered lines
go tool cover -func=pkg_coverage.out | grep -v "100.0%"
```

### Coverage Targets

| Level | Target | Priority |
|-------|--------|----------|
| Critical packages (cmd/*, pkg/security) | 100% | P0 |
| Core packages (pkg/probe, pkg/app) | 95%+ | P1 |
| Supporting packages | 90%+ | P2 |
| Utility packages | 85%+ | P3 |

---

## Writing Tests

### Unit Test Template

```go
package mypackage

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Test Template

```go
// +build integration

package mypackage

import (
    "context"
    "testing"
    "time"
)

func TestIntegration_FullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
    defer cancel()

    // Setup
    server := startTestServer(t, ctx)
    defer server.Shutdown(ctx)

    // Execute test
    // ...

    // Verify results
    // ...
}
```

### Benchmark Test Template

```go
func BenchmarkMyFunction(b *testing.B) {
    input := "test data"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = MyFunction(input)
    }
}

func BenchmarkWithSetup(b *testing.B) {
    // Setup (not timed)
    largeData := generateLargeData()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = ProcessData(largeData)
    }
}
```

### Best Practices

1. **Use Table-Driven Tests**
   - Cover multiple scenarios in one test
   - Easy to add new test cases
   - Improves readability

2. **Test Both Success and Error Paths**
   ```go
   // Good
   func TestFunction(t *testing.T) {
       // Test success case
       result, err := Function(validInput)
       require.NoError(t, err)

       // Test error case
       _, err = Function(invalidInput)
       assert.Error(t, err)
   }
   ```

3. **Use Meaningful Test Names**
   ```go
   // Good
   func TestProbe_CollectHostInfo_WithValidConfig(t *testing.T)

   // Bad
   func TestProbe1(t *testing.T)
   ```

4. **Mock External Dependencies**
   ```go
   type MockDockerClient struct {
       mock.Mock
   }

   func (m *MockDockerClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
       args := m.Called(ctx, options)
       return args.Get(0).([]types.Container), args.Error(1)
   }
   ```

5. **Test Concurrency with -race**
   ```go
   func TestConcurrentAccess(t *testing.T) {
       store := NewStore()

       // Run with: go test -race
       go store.Write("key", "value1")
       go store.Write("key", "value2")
       go store.Read("key")
   }
   ```

---

## CI/CD Integration

### GitHub Actions

The project uses GitHub Actions for continuous integration:

- **On every push**: Run unit tests, linting, and coverage checks
- **On pull request**: Full test suite including integration tests
- **Daily**: Comprehensive test suite with benchmarks
- **On release**: Full validation before deployment

### Pre-commit Hooks

```bash
# Install pre-commit hooks
pre-commit install

# Run manually
pre-commit run --all-files

# Update hooks
pre-commit autoupdate
```

### Coverage Enforcement

- Minimum coverage: **85%** (enforced in CI)
- Target coverage: **100%**
- Pull requests must not decrease coverage
- Critical packages require 95%+ coverage

---

## Troubleshooting

### Common Issues

#### 1. Tests Timing Out

```bash
# Increase timeout
go test -timeout=30m ./...

# Run with verbose output to see where it hangs
go test -v -timeout=5m ./...
```

#### 2. Flaky Tests

```bash
# Run test multiple times to identify flakiness
go test -count=10 ./pkg/probe/...

# Run with race detector
go test -race ./...
```

#### 3. Coverage Not Generated

```bash
# Ensure coverage profile is generated
go test -coverprofile=coverage.out ./...

# Check if file exists
ls -la coverage.out

# Verify coverage data
go tool cover -func=coverage.out
```

#### 4. Integration Tests Failing

```bash
# Check Docker availability
docker info

# Check required services
docker compose ps

# Run with verbose logging
go test -v -tags=integration ./...
```

#### 5. Import Cycle Errors

```bash
# Move shared test utilities to separate package
mkdir testutil
# Move common test helpers there

# Use build tags to exclude from main build
// +build test
```

### Debugging Tests

```bash
# Run single test with verbose output
go test -v -run TestSpecificTest ./pkg/probe/...

# Add debugging output
t.Logf("Debug: variable = %v", variable)

# Use delve debugger
dlv test ./pkg/probe/... -- -test.run TestSpecificTest
```

### Performance Issues

```bash
# Profile CPU usage during tests
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Profile memory usage
go test -memprofile=mem.prof ./...
go tool pprof mem.prof

# Find slow tests
go test -v ./... 2>&1 | grep -E "PASS|FAIL" | sort -k3 -h
```

---

## Appendix

### Useful Commands Cheat Sheet

```bash
# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detection
go test -race ./...

# Benchmarks
go test -bench=. -benchmem ./...

# Specific package
go test -v ./pkg/probe/...

# Integration tests
go test -tags=integration ./...

# Short tests only
go test -short ./...

# With timeout
go test -timeout=10m ./...

# JSON output
go test -json ./...

# Parallel execution
go test -parallel=4 ./...
```

### Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Framework](https://github.com/stretchr/testify)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Advanced Testing in Go](https://about.sourcegraph.com/go/advanced-testing-in-go)
- [Coverage Tool](https://blog.golang.org/cover)

---

**Document Owner:** Engineering Team
**Review Cycle:** Monthly
**Next Review:** 2025-11-18
