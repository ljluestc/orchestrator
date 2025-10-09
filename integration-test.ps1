# Comprehensive System Integration Test
# This script tests all components of the Mesos-Docker Orchestration Platform

Write-Host "Mesos-Docker Orchestration Platform - Integration Test" -ForegroundColor Green
Write-Host "=====================================================" -ForegroundColor Green
Write-Host ""

$testResults = @()

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Url,
        [string]$Method = "GET",
        [string]$Body = $null,
        [string]$ContentType = "application/json"
    )
    
    try {
        if ($Method -eq "GET") {
            $response = Invoke-WebRequest -Uri $Url -TimeoutSec 10
        } else {
            $response = Invoke-WebRequest -Uri $Url -Method $Method -ContentType $ContentType -Body $Body -TimeoutSec 10
        }
        
        $result = @{
            Name = $Name
            Status = "PASS"
            StatusCode = $response.StatusCode
            Message = "Success"
        }
        Write-Host "✓ $Name - Status: $($response.StatusCode)" -ForegroundColor Green
    } catch {
        $result = @{
            Name = $Name
            Status = "FAIL"
            StatusCode = $null
            Message = $_.Exception.Message
        }
        Write-Host "✗ $Name - Error: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    $script:testResults += $result
    return $result
}

# Test 1: App Server Health
Write-Host "1. Testing App Server..." -ForegroundColor Yellow
Test-Endpoint "App Server Health" "http://localhost:8080/health"
Test-Endpoint "App Server Ping" "http://localhost:8080/api/v1/ping"

# Test 2: App Server Agent Management
Write-Host "`n2. Testing Agent Management..." -ForegroundColor Yellow
Test-Endpoint "List Agents" "http://localhost:8080/api/v1/agents/list"

# Test 3: App Server Topology Data
Write-Host "`n3. Testing App Server Topology..." -ForegroundColor Yellow
Test-Endpoint "App Server Topology" "http://localhost:8080/api/v1/query/topology"
Test-Endpoint "App Server Stats" "http://localhost:8080/api/v1/query/stats"

# Test 4: Topology Manager
Write-Host "`n4. Testing Topology Manager..." -ForegroundColor Yellow
Test-Endpoint "Topology Manager Health" "http://localhost:8082/health"
Test-Endpoint "Topology Manager Topology" "http://localhost:8082/api/v1/topology"
Test-Endpoint "Topology Manager Views" "http://localhost:8082/api/v1/views"
Test-Endpoint "Topology Manager Metrics" "http://localhost:8082/api/v1/metrics"

# Test 5: Web UI
Write-Host "`n5. Testing Web UI..." -ForegroundColor Yellow
Test-Endpoint "Web UI" "http://localhost:9090"

# Test 6: Marathon API
Write-Host "`n6. Testing Marathon API..." -ForegroundColor Yellow
Test-Endpoint "Marathon Apps" "http://localhost:8080/v2/apps"

# Test 7: Mesos Master API
Write-Host "`n7. Testing Mesos Master API..." -ForegroundColor Yellow
Test-Endpoint "Mesos Master State" "http://localhost:5050/api/v1"

# Test 8: Probe Agent Data
Write-Host "`n8. Testing Probe Agent Data..." -ForegroundColor Yellow
try {
    $agentsResponse = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/agents/list" -TimeoutSec 10
    $agentsData = $agentsResponse.Content | ConvertFrom-Json
    
    if ($agentsData.count -gt 0) {
        $agentId = $agentsData.agents[0].agent_id
        Test-Endpoint "Latest Report" "http://localhost:8080/api/v1/query/agents/$agentId/latest"
        Test-Endpoint "Time Series Data" "http://localhost:8080/api/v1/query/agents/$agentId/timeseries"
        Write-Host "✓ Found $($agentsData.count) registered agent(s)" -ForegroundColor Green
    } else {
        Write-Host "✗ No agents registered" -ForegroundColor Red
    }
} catch {
    Write-Host "✗ Failed to get agent data: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 9: Data Flow Verification
Write-Host "`n9. Testing Data Flow..." -ForegroundColor Yellow
try {
    # Get topology from app server
    $appTopologyResponse = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/query/topology" -TimeoutSec 10
    $appTopologyData = $appTopologyResponse.Content | ConvertFrom-Json
    
    # Get topology from topology manager
    $topologyManagerResponse = Invoke-WebRequest -Uri "http://localhost:8082/api/v1/topology" -TimeoutSec 10
    $topologyManagerData = $topologyManagerResponse.Content | ConvertFrom-Json
    
    $appNodes = $appTopologyData.topology.nodes.Count
    $appEdges = $appTopologyData.topology.edges.Count
    $tmNodes = $topologyManagerData.nodes.Count
    $tmEdges = $topologyManagerData.edges.Count
    
    Write-Host "App Server Topology: $appNodes nodes, $appEdges edges" -ForegroundColor Cyan
    Write-Host "Topology Manager: $tmNodes nodes, $tmEdges edges" -ForegroundColor Cyan
    
    if ($appNodes -gt 0) {
        Write-Host "✓ App server has topology data" -ForegroundColor Green
    } else {
        Write-Host "✗ App server has no topology data" -ForegroundColor Red
    }
    
    if ($tmNodes -gt 0) {
        Write-Host "✓ Topology manager has data" -ForegroundColor Green
    } else {
        Write-Host "⚠ Topology manager has no data (expected if bridge not working)" -ForegroundColor Yellow
    }
    
} catch {
    Write-Host "✗ Failed to verify data flow: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 10: Process Status
Write-Host "`n10. Testing Process Status..." -ForegroundColor Yellow
$processes = Get-Process | Where-Object { 
    $_.ProcessName -like "*orchestrator*" -or 
    $_.ProcessName -like "*probe*" -or 
    $_.ProcessName -like "*app*" 
}

if ($processes.Count -gt 0) {
    Write-Host "✓ Found $($processes.Count) running processes:" -ForegroundColor Green
    foreach ($proc in $processes) {
        Write-Host "  - $($proc.ProcessName) (PID: $($proc.Id))" -ForegroundColor Cyan
    }
} else {
    Write-Host "✗ No relevant processes found" -ForegroundColor Red
}

# Summary
Write-Host "`n" + "="*60 -ForegroundColor Green
Write-Host "INTEGRATION TEST SUMMARY" -ForegroundColor Green
Write-Host "="*60 -ForegroundColor Green

$passed = ($testResults | Where-Object { $_.Status -eq "PASS" }).Count
$failed = ($testResults | Where-Object { $_.Status -eq "FAIL" }).Count
$total = $testResults.Count

Write-Host "Total Tests: $total" -ForegroundColor White
Write-Host "Passed: $passed" -ForegroundColor Green
Write-Host "Failed: $failed" -ForegroundColor Red

if ($failed -eq 0) {
    Write-Host "`n🎉 ALL TESTS PASSED! System is fully operational." -ForegroundColor Green
} elseif ($passed -gt $failed) {
    Write-Host "`n⚠️  MOSTLY WORKING: $passed/$total tests passed" -ForegroundColor Yellow
} else {
    Write-Host "`n❌ SYSTEM ISSUES: $failed/$total tests failed" -ForegroundColor Red
}

Write-Host "`nFailed Tests:" -ForegroundColor Red
foreach ($test in $testResults | Where-Object { $_.Status -eq "FAIL" }) {
    Write-Host "  - $($test.Name): $($test.Message)" -ForegroundColor Red
}

Write-Host "`nSystem Status:" -ForegroundColor Cyan
Write-Host "  App Server (Port 8080): " -NoNewline
if (($testResults | Where-Object { $_.Name -eq "App Server Health" }).Status -eq "PASS") {
    Write-Host "✓ Running" -ForegroundColor Green
} else {
    Write-Host "✗ Not responding" -ForegroundColor Red
}

Write-Host "  Topology Manager (Port 8082): " -NoNewline
if (($testResults | Where-Object { $_.Name -eq "Topology Manager Health" }).Status -eq "PASS") {
    Write-Host "✓ Running" -ForegroundColor Green
} else {
    Write-Host "✗ Not responding" -ForegroundColor Red
}

Write-Host "  Web UI (Port 9090): " -NoNewline
if (($testResults | Where-Object { $_.Name -eq "Web UI" }).Status -eq "PASS") {
    Write-Host "✓ Running" -ForegroundColor Green
} else {
    Write-Host "✗ Not responding" -ForegroundColor Red
}

Write-Host "  Marathon (Port 8080): " -NoNewline
if (($testResults | Where-Object { $_.Name -eq "Marathon Apps" }).Status -eq "PASS") {
    Write-Host "✓ Running" -ForegroundColor Green
} else {
    Write-Host "✗ Not responding" -ForegroundColor Red
}

Write-Host "  Mesos Master (Port 5050): " -NoNewline
if (($testResults | Where-Object { $_.Name -eq "Mesos Master State" }).Status -eq "PASS") {
    Write-Host "✓ Running" -ForegroundColor Green
} else {
    Write-Host "✗ Not responding" -ForegroundColor Red
}

Write-Host "`nNext Steps:" -ForegroundColor Yellow
Write-Host "1. Open Web UI: http://localhost:9090" -ForegroundColor White
Write-Host "2. View topology data: http://localhost:8082/api/v1/topology" -ForegroundColor White
Write-Host "3. Deploy applications via Marathon: http://localhost:8080/v2/apps" -ForegroundColor White
Write-Host "4. Monitor probe data: http://localhost:8080/api/v1/query/stats" -ForegroundColor White

Write-Host "`nIntegration test completed!" -ForegroundColor Green
