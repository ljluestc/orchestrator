package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ComprehensiveTestSuite manages comprehensive testing
type ComprehensiveTestSuite struct {
	CoverageThreshold float64
	Packages          []string
	Results           map[string]*TestResult
}

// TestResult holds test results for a package
type TestResult struct {
	Package  string
	Coverage float64
	Passed   bool
	Output   string
}

// NewComprehensiveTestSuite creates a new test suite
func NewComprehensiveTestSuite() *ComprehensiveTestSuite {
	return &ComprehensiveTestSuite{
		CoverageThreshold: 100.0,
		Packages: []string{
			"./cmd/app/...",
			"./cmd/probe/...",
			"./cmd/probe-agent/...",
			"./pkg/app/...",
			"./pkg/containerizer/...",
			"./pkg/isolation/...",
			"./pkg/marathon/...",
			"./pkg/mesos/...",
			"./pkg/metrics/...",
			"./pkg/migration/...",
			"./pkg/probe/...",
			"./pkg/scheduler/...",
			"./pkg/security/...",
			"./pkg/topology/...",
			"./pkg/ui/...",
			"./internal/storage/...",
		},
		Results: make(map[string]*TestResult),
	}
}

// RunAllTests executes all tests and generates coverage report
func (suite *ComprehensiveTestSuite) RunAllTests(t *testing.T) error {
	fmt.Println("🚀 Starting Comprehensive Test Suite")
	fmt.Println("=====================================")

	ctx := context.Background()

	for _, pkg := range suite.Packages {
		result := suite.runPackageTests(ctx, pkg)
		suite.Results[pkg] = result

		if !result.Passed {
			t.Errorf("Package %s failed tests", pkg)
		}

		fmt.Printf("\n📦 Package: %s\n", pkg)
		fmt.Printf("   Coverage: %.2f%%\n", result.Coverage)
		fmt.Printf("   Status: %s\n", suite.getStatusEmoji(result.Passed))
	}

	suite.generateSummary(t)
	return nil
}

// runPackageTests runs tests for a specific package
func (suite *ComprehensiveTestSuite) runPackageTests(ctx context.Context, pkg string) *TestResult {
	result := &TestResult{
		Package: pkg,
	}

	coverFile := filepath.Join("coverage", strings.ReplaceAll(pkg, "/", "_")+".out")
	os.MkdirAll("coverage", 0755)

	cmd := exec.CommandContext(ctx, "go", "test", pkg,
		"-coverprofile="+coverFile,
		"-covermode=atomic",
		"-v",
		"-timeout=5m",
	)

	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Passed = false
		return result
	}

	result.Passed = true

	// Parse coverage
	if _, err := os.Stat(coverFile); err == nil {
		coverage := suite.parseCoverage(coverFile)
		result.Coverage = coverage
	}

	return result
}

// parseCoverage extracts coverage percentage from cover file
func (suite *ComprehensiveTestSuite) parseCoverage(file string) float64 {
	cmd := exec.Command("go", "tool", "cover", "-func="+file)
	output, err := cmd.Output()
	if err != nil {
		return 0.0
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				coverage := strings.TrimSuffix(parts[2], "%")
				var pct float64
				fmt.Sscanf(coverage, "%f", &pct)
				return pct
			}
		}
	}
	return 0.0
}

// generateSummary creates test summary report
func (suite *ComprehensiveTestSuite) generateSummary(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	totalPackages := len(suite.Results)
	passedPackages := 0
	totalCoverage := 0.0

	for pkg, result := range suite.Results {
		if result.Passed {
			passedPackages++
		}
		totalCoverage += result.Coverage

		status := "❌ FAIL"
		if result.Passed {
			status = "✅ PASS"
		}

		fmt.Printf("%s %s (%.2f%%)\n", status, pkg, result.Coverage)
	}

	avgCoverage := totalCoverage / float64(totalPackages)

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total Packages: %d\n", totalPackages)
	fmt.Printf("Passed: %d\n", passedPackages)
	fmt.Printf("Failed: %d\n", totalPackages-passedPackages)
	fmt.Printf("Average Coverage: %.2f%%\n", avgCoverage)
	fmt.Println(strings.Repeat("=", 60))

	if avgCoverage < suite.CoverageThreshold {
		t.Errorf("❌ Average coverage %.2f%% is below threshold %.2f%%",
			avgCoverage, suite.CoverageThreshold)
	} else {
		fmt.Printf("✅ Coverage meets threshold (%.2f%% >= %.2f%%)\n",
			avgCoverage, suite.CoverageThreshold)
	}
}

// getStatusEmoji returns status emoji
func (suite *ComprehensiveTestSuite) getStatusEmoji(passed bool) string {
	if passed {
		return "✅ PASSED"
	}
	return "❌ FAILED"
}

// TestComprehensiveSuite is the main test entry point
func TestComprehensiveSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive test suite in short mode")
	}

	suite := NewComprehensiveTestSuite()
	if err := suite.RunAllTests(t); err != nil {
		t.Fatalf("Comprehensive test suite failed: %v", err)
	}
}
