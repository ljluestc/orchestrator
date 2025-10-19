#!/bin/bash
# Automated Test Generator for Missing Coverage
# Analyzes coverage and generates missing tests

set -e

echo "🔍 Analyzing Coverage Gaps and Generating Missing Tests"
echo "========================================================"

GO_BIN="./go/bin/go"
COVERAGE_DIR="coverage"

# Function to analyze coverage for a package
analyze_coverage() {
    local pkg=$1
    local pkg_name=$(echo $pkg | tr '/' '_' | tr '.' '_')
    local cover_file="${COVERAGE_DIR}/${pkg_name}.out"

    if [ ! -f "$cover_file" ]; then
        echo "No coverage file for $pkg, running tests..."
        $GO_BIN test "$pkg" -coverprofile="$cover_file" -covermode=atomic || true
    fi

    if [ -f "$cover_file" ]; then
        echo ""
        echo "📦 Package: $pkg"
        echo "----------------------------------------"

        # Show uncovered functions
        $GO_BIN tool cover -func="$cover_file" | grep -E ":[0-9]+:\s+[a-zA-Z]" | while read -r line; do
            coverage=$(echo "$line" | awk '{print $3}' | sed 's/%//')
            if (( $(echo "$coverage < 100" | bc -l) )); then
                echo "⚠️  $line"
            fi
        done

        # Overall coverage
        total_coverage=$($GO_BIN tool cover -func="$cover_file" | tail -1 | awk '{print $3}')
        echo ""
        echo "Total Coverage: $total_coverage"
    fi
}

# Packages to analyze
PACKAGES=(
    "./cmd/app"
    "./cmd/probe"
    "./cmd/probe-agent"
    "./pkg/app"
    "./pkg/containerizer"
    "./pkg/isolation"
    "./pkg/marathon"
    "./pkg/mesos"
    "./pkg/metrics"
    "./pkg/migration"
    "./pkg/probe"
    "./pkg/scheduler"
    "./pkg/security"
    "./pkg/topology"
    "./pkg/ui"
    "./internal/storage"
)

# Analyze all packages
for pkg in "${PACKAGES[@]}"; do
    analyze_coverage "$pkg"
done

echo ""
echo "========================================================"
echo "✅ Coverage analysis complete!"
echo "========================================================"
