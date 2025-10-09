# Simple Integration Test
Write-Host "Mesos-Docker Orchestration Platform - Integration Test" -ForegroundColor Green
Write-Host "=====================================================" -ForegroundColor Green

# Test App Server
Write-Host "`nTesting App Server..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 10
    Write-Host "✓ App Server Health: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "✗ App Server Health: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Topology Manager
Write-Host "`nTesting Topology Manager..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8082/health" -TimeoutSec 10
    Write-Host "✓ Topology Manager Health: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "✗ Topology Manager Health: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Web UI
Write-Host "`nTesting Web UI..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:9090" -TimeoutSec 10
    Write-Host "✓ Web UI: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "✗ Web UI: $($_.Exception.Message)" -ForegroundColor Red
}

# Test App Server Topology
Write-Host "`nTesting App Server Topology..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/query/topology" -TimeoutSec 10
    $data = $response.Content | ConvertFrom-Json
    Write-Host "✓ App Server Topology: $($data.topology.nodes.Count) nodes, $($data.topology.edges.Count) edges" -ForegroundColor Green
} catch {
    Write-Host "✗ App Server Topology: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Topology Manager Data
Write-Host "`nTesting Topology Manager Data..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8082/api/v1/topology" -TimeoutSec 10
    $data = $response.Content | ConvertFrom-Json
    Write-Host "✓ Topology Manager: $($data.nodes.Count) nodes, $($data.edges.Count) edges" -ForegroundColor Green
} catch {
    Write-Host "✗ Topology Manager: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Agents
Write-Host "`nTesting Agents..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/agents/list" -TimeoutSec 10
    $data = $response.Content | ConvertFrom-Json
    Write-Host "✓ Registered Agents: $($data.count)" -ForegroundColor Green
} catch {
    Write-Host "✗ Agents: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Marathon
Write-Host "`nTesting Marathon..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/v2/apps" -TimeoutSec 10
    Write-Host "✓ Marathon API: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "✗ Marathon API: $($_.Exception.Message)" -ForegroundColor Red
}

# Test Mesos Master
Write-Host "`nTesting Mesos Master..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:5050/api/v1" -TimeoutSec 10
    Write-Host "✓ Mesos Master: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "✗ Mesos Master: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n" + "="*50 -ForegroundColor Green
Write-Host "INTEGRATION TEST COMPLETED" -ForegroundColor Green
Write-Host "="*50 -ForegroundColor Green

Write-Host "`nSystem URLs:" -ForegroundColor Cyan
Write-Host "  Web UI: http://localhost:9090" -ForegroundColor White
Write-Host "  App Server API: http://localhost:8080/api/v1" -ForegroundColor White
Write-Host "  Topology Manager: http://localhost:8082/api/v1" -ForegroundColor White
Write-Host "  Marathon API: http://localhost:8080/v2" -ForegroundColor White
Write-Host "  Mesos Master: http://localhost:5050/api/v1" -ForegroundColor White

Write-Host "`nThe Mesos-Docker Orchestration Platform is running!" -ForegroundColor Green
