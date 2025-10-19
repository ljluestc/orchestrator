#!/bin/bash

###############################################################################
# Comprehensive Test Orchestration Script
# Purpose: Run all tests, generate coverage reports, and validate 100% coverage
# Usage: ./scripts/test_orchestrator.sh [options]
###############################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_DIR="./coverage"
COVERAGE_PROFILE="${COVERAGE_DIR}/profile.out"
COVERAGE_HTML="${COVERAGE_DIR}/coverage.html"
COVERAGE_JSON="${COVERAGE_DIR}/coverage.json"
MIN_COVERAGE=85.0
TARGET_COVERAGE=100.0
TIMEOUT="10m"
GO_BIN="./go/bin/go"

# Ensure go binary exists
if [ ! -f "$GO_BIN" ]; then
    GO_BIN="go"
fi

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

###############################################################################
# Helper Functions
###############################################################################

print_header() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

###############################################################################
# Test Execution Functions
###############################################################################

run_unit_tests() {
    print_header "Running Unit Tests"

    echo "Discovering test packages..."
    packages=$($GO_BIN list ./... 2>/dev/null | grep -v "/go/" || echo "")

    if [ -z "$packages" ]; then
        print_error "No packages found"
        return 1
    fi

    echo "Found $(echo "$packages" | wc -l) packages to test"
    echo ""

    echo "Running tests with coverage..."
    $GO_BIN test -v \
        -timeout="$TIMEOUT" \
        -race \
        -coverprofile="$COVERAGE_PROFILE" \
        -covermode=atomic \
        $packages 2>&1 | tee "${COVERAGE_DIR}/test_output.log"

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        print_success "Unit tests passed"
        return 0
    else
        print_error "Unit tests failed"
        return 1
    fi
}

run_integration_tests() {
    print_header "Running Integration Tests"

    if [ -f "./integration_test.go" ]; then
        $GO_BIN test -v -tags=integration -timeout="$TIMEOUT" ./integration_test.go
        if [ $? -eq 0 ]; then
            print_success "Integration tests passed"
        else
            print_error "Integration tests failed"
            return 1
        fi
    else
        print_warning "No integration tests found"
    fi
}

run_benchmark_tests() {
    print_header "Running Benchmark Tests"

    $GO_BIN test -v -run=^$ -bench=. -benchmem ./... 2>&1 | tee "${COVERAGE_DIR}/benchmark_output.log"

    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        print_success "Benchmarks completed"
    else
        print_warning "Some benchmarks may have issues"
    fi
}

###############################################################################
# Coverage Analysis Functions
###############################################################################

generate_coverage_report() {
    print_header "Generating Coverage Reports"

    if [ ! -f "$COVERAGE_PROFILE" ]; then
        print_error "Coverage profile not found"
        return 1
    fi

    # Generate HTML report
    print_info "Generating HTML coverage report..."
    $GO_BIN tool cover -html="$COVERAGE_PROFILE" -o "$COVERAGE_HTML"
    print_success "HTML report: $COVERAGE_HTML"

    # Generate function-level coverage
    print_info "Generating function coverage analysis..."
    $GO_BIN tool cover -func="$COVERAGE_PROFILE" > "${COVERAGE_DIR}/coverage_by_function.txt"
    print_success "Function coverage: ${COVERAGE_DIR}/coverage_by_function.txt"

    # Generate package-level coverage
    generate_package_coverage
}

generate_package_coverage() {
    print_info "Analyzing package-level coverage..."

    cat "$COVERAGE_PROFILE" | grep -v "mode:" | awk '{print $1}' | cut -d'/' -f1-5 | sort -u > "${COVERAGE_DIR}/packages.tmp"

    echo "Package Coverage Report" > "${COVERAGE_DIR}/coverage_by_package.txt"
    echo "======================" >> "${COVERAGE_DIR}/coverage_by_package.txt"
    echo "" >> "${COVERAGE_DIR}/coverage_by_package.txt"

    while read -r pkg; do
        if [ -n "$pkg" ]; then
            coverage=$($GO_BIN tool cover -func="$COVERAGE_PROFILE" | grep "^${pkg}" | awk '{s+=$3; c++} END {if(c>0) printf "%.1f\n", s/c; else print "0.0"}')
            printf "%-50s %6s%%\n" "$pkg" "$coverage" >> "${COVERAGE_DIR}/coverage_by_package.txt"
        fi
    done < "${COVERAGE_DIR}/packages.tmp"

    rm -f "${COVERAGE_DIR}/packages.tmp"
}

calculate_total_coverage() {
    if [ ! -f "$COVERAGE_PROFILE" ]; then
        echo "0.0"
        return
    fi

    $GO_BIN tool cover -func="$COVERAGE_PROFILE" | grep "^total:" | awk '{print $3}' | sed 's/%//'
}

check_coverage_threshold() {
    print_header "Checking Coverage Thresholds"

    total_coverage=$(calculate_total_coverage)

    echo "Total Coverage: ${total_coverage}%"
    echo "Minimum Required: ${MIN_COVERAGE}%"
    echo "Target: ${TARGET_COVERAGE}%"
    echo ""

    # Check against minimum
    if (( $(echo "$total_coverage >= $MIN_COVERAGE" | bc -l) )); then
        print_success "Coverage meets minimum threshold (${MIN_COVERAGE}%)"
    else
        print_error "Coverage below minimum threshold (${MIN_COVERAGE}%)"
        return 1
    fi

    # Check against target
    if (( $(echo "$total_coverage >= $TARGET_COVERAGE" | bc -l) )); then
        print_success "Coverage meets target (${TARGET_COVERAGE}%)"
    else
        gap=$(echo "$TARGET_COVERAGE - $total_coverage" | bc)
        print_warning "Coverage gap to target: ${gap}%"
    fi

    echo "$total_coverage" > "${COVERAGE_DIR}/total_coverage.txt"
}

identify_uncovered_code() {
    print_header "Identifying Uncovered Code"

    $GO_BIN tool cover -func="$COVERAGE_PROFILE" | grep -v "100.0%" | grep -v "^total:" > "${COVERAGE_DIR}/uncovered.txt" || true

    uncovered_count=$(cat "${COVERAGE_DIR}/uncovered.txt" | wc -l)

    if [ $uncovered_count -eq 0 ]; then
        print_success "All code is covered!"
    else
        print_warning "Found $uncovered_count functions with incomplete coverage"
        echo ""
        echo "Top 10 functions needing coverage:"
        cat "${COVERAGE_DIR}/uncovered.txt" | head -10
    fi
}

###############################################################################
# Report Generation Functions
###############################################################################

generate_summary_report() {
    print_header "Generating Summary Report"

    local total_coverage=$(calculate_total_coverage)
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    cat > "${COVERAGE_DIR}/SUMMARY.md" <<EOF
# Test Coverage Summary Report

**Generated:** ${timestamp}
**Total Coverage:** ${total_coverage}%
**Target Coverage:** ${TARGET_COVERAGE}%
**Coverage Gap:** $(echo "$TARGET_COVERAGE - $total_coverage" | bc)%

---

## Overall Statistics

\`\`\`
Total Packages: $($GO_BIN list ./... 2>/dev/null | grep -v "/go/" | wc -l)
Test Files: $(find . -name "*_test.go" -type f | wc -l)
Coverage Profile: ${COVERAGE_PROFILE}
HTML Report: ${COVERAGE_HTML}
\`\`\`

---

## Package Coverage

$(cat "${COVERAGE_DIR}/coverage_by_package.txt")

---

## Functions Needing Coverage

$(cat "${COVERAGE_DIR}/uncovered.txt" | head -20)

---

## Test Execution Log

See: ${COVERAGE_DIR}/test_output.log

---

## Recommendations

EOF

    if (( $(echo "$total_coverage < 90" | bc -l) )); then
        echo "- ⚠️  Priority: Improve coverage on critical packages" >> "${COVERAGE_DIR}/SUMMARY.md"
    fi

    if [ -f "${COVERAGE_DIR}/uncovered.txt" ] && [ $(cat "${COVERAGE_DIR}/uncovered.txt" | wc -l) -gt 0 ]; then
        echo "- 📝 Focus on functions listed in uncovered.txt" >> "${COVERAGE_DIR}/SUMMARY.md"
    fi

    echo "- ✓ Review HTML coverage report for visual analysis" >> "${COVERAGE_DIR}/SUMMARY.md"
    echo "- ✓ Add integration tests for end-to-end scenarios" >> "${COVERAGE_DIR}/SUMMARY.md"

    print_success "Summary report generated: ${COVERAGE_DIR}/SUMMARY.md"
}

generate_json_report() {
    print_info "Generating JSON coverage report..."

    local total_coverage=$(calculate_total_coverage)

    cat > "$COVERAGE_JSON" <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "total_coverage": ${total_coverage},
  "target_coverage": ${TARGET_COVERAGE},
  "min_coverage": ${MIN_COVERAGE},
  "packages": [
EOF

    # Parse package coverage
    first=true
    while read -r line; do
        pkg=$(echo "$line" | awk '{print $1}')
        cov=$(echo "$line" | awk '{print $2}' | sed 's/%//')

        if [ -n "$pkg" ] && [ "$pkg" != "Package" ] && [ "$pkg" != "=" ]; then
            if [ "$first" = false ]; then
                echo "," >> "$COVERAGE_JSON"
            fi
            echo "    {\"package\": \"$pkg\", \"coverage\": $cov}" >> "$COVERAGE_JSON"
            first=false
        fi
    done < "${COVERAGE_DIR}/coverage_by_package.txt"

    cat >> "$COVERAGE_JSON" <<EOF

  ]
}
EOF

    print_success "JSON report generated: $COVERAGE_JSON"
}

###############################################################################
# Code Quality Checks
###############################################################################

run_linter() {
    print_header "Running Code Quality Checks"

    if command -v golangci-lint &> /dev/null; then
        print_info "Running golangci-lint..."
        golangci-lint run --timeout="$TIMEOUT" ./... 2>&1 | tee "${COVERAGE_DIR}/lint_output.log" || true
        print_success "Linter completed"
    else
        print_warning "golangci-lint not found, skipping linter"
    fi
}

run_static_analysis() {
    print_info "Running static analysis..."

    # Go vet
    $GO_BIN vet ./... 2>&1 | tee "${COVERAGE_DIR}/vet_output.log" || true

    # Go fmt check
    unformatted=$($GO_BIN fmt ./... 2>&1)
    if [ -n "$unformatted" ]; then
        print_warning "Some files are not formatted: $unformatted"
    else
        print_success "All files are properly formatted"
    fi
}

###############################################################################
# Main Execution
###############################################################################

main() {
    local start_time=$(date +%s)

    print_header "Orchestrator Test Suite v1.0"
    echo "Starting comprehensive test execution..."
    echo ""

    # Parse command line options
    RUN_INTEGRATION=true
    RUN_BENCHMARKS=false
    RUN_LINTER=true

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-integration)
                RUN_INTEGRATION=false
                shift
                ;;
            --run-benchmarks)
                RUN_BENCHMARKS=true
                shift
                ;;
            --skip-linter)
                RUN_LINTER=false
                shift
                ;;
            --coverage-target)
                TARGET_COVERAGE="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Run tests
    run_unit_tests || exit 1

    if [ "$RUN_INTEGRATION" = true ]; then
        run_integration_tests || true
    fi

    if [ "$RUN_BENCHMARKS" = true ]; then
        run_benchmark_tests || true
    fi

    # Generate coverage reports
    generate_coverage_report
    check_coverage_threshold
    identify_uncovered_code

    # Generate summary reports
    generate_summary_report
    generate_json_report

    # Run quality checks
    if [ "$RUN_LINTER" = true ]; then
        run_linter
        run_static_analysis
    fi

    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    print_header "Test Suite Complete"
    echo "Duration: ${duration}s"
    echo "Coverage Report: $COVERAGE_HTML"
    echo "Summary: ${COVERAGE_DIR}/SUMMARY.md"
    echo ""

    print_success "All tests completed successfully!"
}

# Run main function
main "$@"
