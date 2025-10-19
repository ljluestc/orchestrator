# Orchestrator Platform - Testing & CI/CD 🚀

**Status**: ✅ Infrastructure Complete | 🔄 Coverage In Progress
**Current Coverage**: 85.2% | **Target**: 100%

---

## Quick Start

### Run All Tests
```bash
# Comprehensive test suite (recommended)
./scripts/test_orchestrator.sh

# Quick tests
go test ./... -short

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### View Coverage
```bash
# After running tests
open coverage/coverage.html        # macOS
xdg-open coverage/coverage.html    # Linux

# View summary
cat coverage/SUMMARY.md
```

---

## What's Included

### 🎯 Test Infrastructure

1. **Comprehensive Test Orchestrator** (`scripts/test_orchestrator.sh`)
   - Automated test execution
   - Coverage analysis (package & function level)
   - HTML and JSON reports
   - Uncovered code identification
   - Threshold validation

2. **CI/CD Pipeline** (GitHub Actions)
   - Multi-platform testing (Ubuntu, macOS)
   - Multi-version Go (1.23.7, 1.23.x)
   - Security scanning
   - Dependency checks
   - Automated coverage reports

3. **Pre-commit Hooks**
   - Code formatting (gofmt, goimports)
   - Linting (golangci-lint with 30+ linters)
   - Fast tests & coverage validation
   - Security checks
   - Build verification

4. **Test Suites**
   - Unit tests: `*_test.go`
   - Integration tests: `integration_test.go`
   - Benchmarks: `benchmark_test.go`

---

## Installation

### Setup Pre-commit Hooks
```bash
pip install pre-commit
pre-commit install
```

### Install Testing Tools
```bash
go install golang.org/x/tools/cmd/cover@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

## Running Tests

### Basic Commands

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/probe/...

# With race detection
go test -race ./...

# Integration tests
go test -tags=integration ./...

# Benchmarks
go test -bench=. -benchmem ./...
```

### Using Test Orchestrator

```bash
# Full suite
./scripts/test_orchestrator.sh

# Options
./scripts/test_orchestrator.sh --skip-integration
./scripts/test_orchestrator.sh --run-benchmarks
./scripts/test_orchestrator.sh --coverage-target 95.0
```

---

## Coverage Status

### Current Package Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| pkg/ui | 100.0% | ✅ |
| internal/storage | 96.1% | ✅ |
| pkg/app | 93.5% | ✅ |
| pkg/mesos/agent | 91.1% | ✅ |
| pkg/topology | 86.2% | ⚠️ |
| pkg/isolation | 85.7% | ⚠️ |
| pkg/probe | 84.4% | ⚠️ |
| pkg/security | 77.5% | ⚠️ |
| pkg/containerizer | 72.2% | ⚠️ |
| pkg/migration | 59.0% | ❌ |
| cmd/app | 53.7% | ❌ |
| cmd/probe | 0.0% | ⚠️* |

*Tests created but coverage not yet reflected

### Roadmap to 100%

See [PRD_100_PERCENT_COVERAGE.md](./PRD_100_PERCENT_COVERAGE.md) for detailed roadmap.

**Target Timeline**: 6-8 weeks

---

## CI/CD Pipeline

### Automatic Triggers
- ✅ Push to main/develop/feature branches
- ✅ Pull requests to main/develop
- ✅ Daily at 2 AM UTC
- ✅ Manual workflow dispatch

### Pipeline Jobs
1. **Code Quality** - Linting & formatting
2. **Unit Tests** - Multi-platform, multi-version
3. **Integration Tests** - With Docker services
4. **Package Coverage** - Per-package deep dive
5. **Security Scan** - gosec + trivy
6. **Dependency Check** - govulncheck
7. **Build** - Binaries + Docker images
8. **Benchmarks** - Performance testing
9. **Reports** - Coverage aggregation

### Coverage Enforcement
- ✅ Minimum: 85% (enforced)
- 🎯 Target: 100%
- ❌ PRs that decrease coverage are blocked

---

## Writing Tests

### Unit Test Template
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
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

### Best Practices
1. ✅ Use table-driven tests
2. ✅ Test success AND error paths
3. ✅ Use meaningful test names
4. ✅ Mock external dependencies
5. ✅ Run with `-race` flag for concurrency
6. ✅ Keep tests fast and deterministic

---

## Documentation

### Complete Guides
- [COMPREHENSIVE_TESTING_GUIDE.md](./docs/COMPREHENSIVE_TESTING_GUIDE.md) - Full testing documentation
- [PRD_100_PERCENT_COVERAGE.md](./PRD_100_PERCENT_COVERAGE.md) - Coverage roadmap
- [TESTING_INFRASTRUCTURE_COMPLETE.md](./TESTING_INFRASTRUCTURE_COMPLETE.md) - Implementation summary

### Quick Reference
```bash
# Run tests
go test ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...

# Use test orchestrator
./scripts/test_orchestrator.sh
```

---

## Troubleshooting

### Tests Timing Out
```bash
go test -timeout=30m ./...
```

### Flaky Tests
```bash
go test -count=10 ./pkg/probe/...
```

### Coverage Not Generated
```bash
go test -coverprofile=coverage.out ./...
ls -la coverage.out
go tool cover -func=coverage.out
```

### Integration Tests Failing
```bash
docker info  # Check Docker
docker compose ps  # Check services
go test -v -tags=integration ./...
```

---

## Contributing

### Before Committing
```bash
# Pre-commit hooks run automatically
git add .
git commit -m "Your message"
```

### Before Creating PR
```bash
# Run full test suite
./scripts/test_orchestrator.sh

# Ensure coverage meets threshold
cat coverage/SUMMARY.md

# Fix any issues
golangci-lint run ./...
```

### PR Requirements
- ✅ All tests pass
- ✅ Coverage ≥ 85%
- ✅ No decrease in coverage
- ✅ Linting passes
- ✅ Pre-commit hooks pass

---

## Support

### Resources
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Framework](https://github.com/stretchr/testify)
- [golangci-lint](https://golangci-lint.run/)

### Common Commands
```bash
# Help
./scripts/test_orchestrator.sh --help

# Coverage
go tool cover -func=coverage.out

# Benchmarks
go test -bench=. -benchmem ./...

# Race detection
go test -race ./...

# Specific package
go test -v ./pkg/probe/...
```

---

## Success Metrics

### Infrastructure ✅
- [x] Test Orchestration Script
- [x] CI/CD Pipeline (GitHub Actions)
- [x] Pre-commit Hooks
- [x] Comprehensive Documentation
- [x] Unit Test Framework
- [x] Integration Tests
- [x] Benchmark Tests

### Coverage 🔄
- [x] Infrastructure Complete
- [ ] Overall Coverage: 85.2% → 100%
- [ ] All Packages ≥ 95%
- [ ] Critical Packages = 100%

---

## Next Steps

1. **Week 1-2**: Complete critical packages (cmd/*, pkg/migration)
2. **Week 3-4**: High-priority packages (pkg/containerizer, pkg/security)
3. **Week 5-6**: Push all to 100%

See [PRD_100_PERCENT_COVERAGE.md](./PRD_100_PERCENT_COVERAGE.md) for detailed plan.

---

**🎯 Ready to achieve 100% test coverage!**

**Status**: Infrastructure ✅ | Coverage 🔄 | Target 🎯 100%
