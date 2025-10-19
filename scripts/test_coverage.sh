#!/bin/bash
# Test coverage script for orchestrator project
# This script runs tests on all packages and generates coverage reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}===========================================${NC}"
echo -e "${BLUE}  Orchestrator Test Coverage Analysis${NC}"
echo -e "${BLUE}===========================================${NC}"
echo ""

# Create coverage directory
mkdir -p coverage

# Test packages
PACKAGES="./pkg/... ./cmd/... ./internal/..."

echo -e "${YELLOW}Step 1: Running unit tests...${NC}"
./go/bin/go test $PACKAGES \
    -coverprofile=coverage/coverage.out \
    -covermode=atomic \
    -v | tee coverage/test.log

echo ""
echo -e "${YELLOW}Step 2: Generating HTML coverage report...${NC}"
./go/bin/go tool cover -html=coverage/coverage.out -o coverage/coverage.html

echo ""
echo -e "${YELLOW}Step 3: Calculating coverage by package...${NC}"
./go/bin/go tool cover -func=coverage/coverage.out > coverage/coverage_by_func.txt

# Calculate total coverage
TOTAL_COVERAGE=$(./go/bin/go tool cover -func=coverage/coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')

echo ""
echo -e "${BLUE}===========================================${NC}"
echo -e "${GREEN}Total Coverage: ${TOTAL_COVERAGE}%${NC}"
echo -e "${BLUE}===========================================${NC}"

# Parse coverage and find low-coverage files
echo ""
echo -e "${YELLOW}Packages with < 80% coverage:${NC}"
./go/bin/go tool cover -func=coverage/coverage.out | \
    awk -F'[:\t]' '
        /total:/ {
            if ($NF+0 < 80.0) {
                printf "  %-50s %s\n", $1, $NF
            }
        }
    '

echo ""
echo -e "${YELLOW}Functions with 0% coverage:${NC}"
./go/bin/go tool cover -func=coverage/coverage.out | \
    awk '{
        if ($3 == "0.0%") {
            print "  " $1 ":" $2
        }
    }' | head -20

echo ""
echo -e "${BLUE}Coverage reports generated in:${NC}"
echo -e "  - coverage/coverage.out (machine-readable)"
echo -e "  - coverage/coverage.html (visual report)"
echo -e "  - coverage/coverage_by_func.txt (detailed breakdown)"
echo ""

# Check if coverage meets threshold
THRESHOLD=85
if (( $(echo "$TOTAL_COVERAGE >= $THRESHOLD" | bc -l) )); then
    echo -e "${GREEN}✓ Coverage ${TOTAL_COVERAGE}% meets threshold of ${THRESHOLD}%${NC}"
    exit 0
else
    echo -e "${RED}✗ Coverage ${TOTAL_COVERAGE}% is below threshold of ${THRESHOLD}%${NC}"
    exit 1
fi
