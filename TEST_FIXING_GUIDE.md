# Test Fixing Guide for 100% Coverage

## Current Status (as of completion)

### ✅ Fixed Packages
- **pkg/containerizer**: Fixed missing `fmt` import
- **pkg/marathon**: Fixed task ID generation (`strconv.Itoa` instead of `string(rune)`)
- **pkg/mesos**: Fixed routing tests to use router instead of direct handler calls, fixed `http.ErrServerClosed` handling

### ⚠️ Packages Still Needing Fixes
- **pkg/migration**: Multiple test failures (similar patterns to mesos)
- **pkg/probe**: Test failures
- **pkg/security**: Test failures

### 📊 Coverage Status
| Package | Current Coverage | Tests Status |
|---------|-----------------|--------------|
| root | 16.0% | Passing |
| cmd/app | 85.4% | Passing |
| cmd/probe | 0.0% | Passing |
| cmd/probe-agent | 52.3% | Passing |
| internal/storage | 96.1% | Passing |
| pkg/app | 93.4% | Passing |
| pkg/containerizer | 72.2% | Passing |
| pkg/isolation | 85.7% | Passing |
| pkg/marathon | N/A | ⚠️ Failing (all tests pass individually) |
| pkg/mesos | ~85% | ✅ Fixed |
| pkg/metrics | 100.0% | ✅ Passing |
| pkg/migration | ~75% | ⚠️ Failing |
| pkg/probe | ~80% | ⚠️ Failing |
| pkg/scheduler | 96.2% | Passing |
| pkg/security | ~70% | ⚠️ Failing |
| pkg/topology | 85.9% | Passing |
| pkg/ui | 100.0% | ✅ Passing |

## Common Test Fix Patterns

### Pattern 1: HTTP Handler Tests with URL Parameters

**Problem**: Tests calling handlers directly without router fail to extract URL parameters.

**Solution**:
```go
// ❌ Wrong
req := httptest.NewRequest("GET", "/api/v1/tasks/task-1", nil)
handler(rr, req)  // mux.Vars(r) returns empty map

// ✅ Correct
router := setupRoutes()
req := httptest.NewRequest("GET", "/api/v1/tasks/task-1", nil)
router.ServeHTTP(rr, req)  // Router sets vars in context
```

**Files Fixed**:
- `pkg/mesos/master_test.go`: TestMaster_HandleGetTask, TestMaster_HandleKillTask

### Pattern 2: Server Start/Stop Tests

**Problem**: `http.ErrServerClosed` treated as error when it's expected behavior.

**Solution**:
```go
// ❌ Wrong
select {
case err := <-errChan:
    assert.NoError(t, err)  // Fails with "http: Server closed"
}

// ✅ Correct
select {
case err := <-errChan:
    if err != nil && err != http.ErrServerClosed {
        assert.NoError(t, err)
    }
}
```

**Files Fixed**:
- `pkg/mesos/master_test.go`: TestMaster_StartStop
- `pkg/migration/zookeeper_test.go`: TestMigrationManager_StartStop

### Pattern 3: Missing Test Data for Route Tests

**Problem**: Routes return 404 because resources don't exist, not because route is missing.

**Solution**:
```go
// ✅ Add test data before testing routes
agent := &AgentInfo{ID: "agent-1", ...}
master.RegisterAgent(agent)

framework := &Framework{ID: "framework-1", ...}
master.RegisterFramework(framework)

// Now test routes
router.ServeHTTP(rr, req)  // Won't get 404 from missing data
```

**Files Fixed**:
- `pkg/mesos/master_test.go`: TestMaster_SetupRoutes

### Pattern 4: Type Mismatches

**Problem**: Incorrect assumptions about data structure types.

**Solution**: Check actual type definitions
```go
// ❌ Wrong assumption
master.State.Offers["offer-1"] = offer  // Offers is a map

// ✅ Correct (Offers is a slice)
master.State.Offers = append(master.State.Offers, offer)
```

## Systematic Fix Approach

### For Each Failing Package:

1. **Identify Failing Tests**
   ```bash
   ./go/bin/go test -json github.com/ljluestc/orchestrator/pkg/PACKAGE 2>&1 | grep '"Action":"fail"'
   ```

2. **Run Individual Test with Details**
   ```bash
   ./go/bin/go test -v github.com/ljluestc/orchestrator/pkg/PACKAGE -run TestName
   ```

3. **Apply Common Patterns**
   - Check for direct handler calls → Use router
   - Check for http.ErrServerClosed → Handle explicitly
   - Check for missing test data → Add setup
   - Check for type assumptions → Verify actual types

4. **Verify Fix**
   ```bash
   ./go/bin/go test github.com/ljluestc/orchestrator/pkg/PACKAGE
   ```

## Remaining Work for 100% Coverage

### pkg/migration (~10 failing tests)
**Similar issues to mesos**:
- Route tests need setup data
- Handler tests need router
- StartStop test fixed, but others remain

**Next Steps**:
1. Run: `./go/bin/go test -v github.com/ljluestc/orchestrator/pkg/migration -run TestNewSyncEngine`
2. Fix similar to mesos patterns
3. Repeat for other failing tests

### pkg/probe (~5 failing tests)
**Likely issues**:
- Same router/handler pattern
- Missing test data

**Next Steps**:
1. Identify specific failures
2. Apply patterns 1-3

### pkg/security (~3 failing tests)
**Likely issues**:
- Authentication/authorization test setup
- Mock dependencies

**Next Steps**:
1. Check what's failing
2. Add necessary mocks/setup

### Coverage Improvements Needed

| Package | Current | Target | Lines to Cover |
|---------|---------|--------|----------------|
| root | 16% | 100% | ~100 lines |
| cmd/probe | 0% | 100% | ~50 lines |
| cmd/probe-agent | 52% | 100% | ~30 lines |
| pkg/containerizer | 72% | 100% | ~50 lines |
| cmd/app | 85% | 100% | ~20 lines |
| pkg/isolation | 86% | 100% | ~15 lines |
| pkg/topology | 86% | 100% | ~30 lines |
| pkg/app | 93% | 100% | ~20 lines |
| pkg/scheduler | 96% | 100% | ~10 lines |
| internal/storage | 96% | 100% | ~10 lines |

**Total new test code needed**: ~300-400 lines

## Tools Created

### `/home/calelin/dev/orchestrator/scripts/achieve_100_percent_coverage.sh`
- Identifies failing packages
- Analyzes coverage
- Generates reports

### `/home/calelin/dev/orchestrator/coverage_analysis/`
- Detailed per-package coverage files
- HTML coverage report
- Summary statistics

## Recommendations

### Immediate (1-2 hours)
1. Fix remaining pkg/migration tests (10 tests, ~30 min)
2. Fix pkg/probe tests (~20 min)
3. Fix pkg/security tests (~20 min)
4. Debug pkg/marathon exit code issue (~20 min)

### Short-term (2-4 hours)
1. Add tests for uncovered lines in high-value packages:
   - internal/storage (4% to go)
   - pkg/scheduler (4% to go)
   - pkg/app (7% to go)

### Medium-term (4-8 hours)
1. Add comprehensive tests for low-coverage packages:
   - cmd/probe (0% → 100%)
   - root package (16% → 100%)
   - cmd/probe-agent (52% → 100%)
   - pkg/containerizer (72% → 100%)

2. Create performance benchmarks
3. Create integration test suite
4. Write ARCHITECTURE.md

### CI/CD Updates
1. Update `.github/workflows/*.yml` to enforce 100% coverage
2. Add coverage badges to README
3. Set up automatic coverage reporting

## Quick Reference Commands

```bash
# Run all tests
./go/bin/go test ./...

# Run with coverage
./go/bin/go test -cover ./...

# Generate coverage profile
./go/bin/go test -coverprofile=coverage.out ./...

# View coverage in browser
./go/bin/go tool cover -html=coverage.out

# Run specific package
./go/bin/go test -v github.com/ljluestc/orchestrator/pkg/PACKAGE

# Run specific test
./go/bin/go test -v -run TestName github.com/ljluestc/orchestrator/pkg/PACKAGE

# Check failing tests
./go/bin/go test -json ./... 2>&1 | grep '"Action":"fail"'

# Coverage by package
./go/bin/go test -cover ./... | grep -E "coverage:|ok|FAIL"
```

## Success Criteria

- [ ] All test packages pass
- [ ] Overall coverage >= 100%
- [ ] Each package >= 100%
- [ ] CI/CD enforces coverage
- [ ] Pre-commit hooks pass
- [ ] Integration tests pass
- [ ] Performance benchmarks created
- [ ] Documentation complete
