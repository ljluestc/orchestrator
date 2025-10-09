# Bridge script to connect app server to topology manager
# This script polls the app server for topology data and pushes it to the topology manager

Write-Host "Starting topology bridge..." -ForegroundColor Green

$appServerURL = "http://localhost:8080"
$topologyManagerURL = "http://localhost:8082"
$pollInterval = 30 # seconds

function Get-AppServerTopology {
    try {
        $response = Invoke-WebRequest -Uri "$appServerURL/api/v1/query/topology" -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            return $response.Content | ConvertFrom-Json
        }
    } catch {
        Write-Host "Failed to fetch from app server: $($_.Exception.Message)" -ForegroundColor Red
    }
    return $null
}

function Push-ToTopologyManager($topologyData) {
    if ($topologyData -eq $null) { return }
    
    # Process nodes
    foreach ($nodeId in $topologyData.topology.nodes.PSObject.Properties.Name) {
        $node = $topologyData.topology.nodes.$nodeId
        
        $nodePayload = @{
            id = $node.id
            type = $node.type
            name = $node.name
            metadata = @{
                source = "bridge"
                original_metadata = $node.metadata
            }
        }
        
        if ($node.parent_id) {
            $nodePayload.metadata.parent_id = $node.parent_id
        }
        
        try {
            $jsonPayload = $nodePayload | ConvertTo-Json -Depth 3
            $response = Invoke-WebRequest -Uri "$topologyManagerURL/api/v1/topology/nodes" -Method POST -ContentType "application/json" -Body $jsonPayload -TimeoutSec 10
            if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 201) {
                Write-Host "Pushed node: $nodeId" -ForegroundColor Green
            }
        } catch {
            Write-Host "Failed to push node $nodeId`: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
    
    # Process edges
    foreach ($edgeId in $topologyData.topology.edges.PSObject.Properties.Name) {
        $edge = $topologyData.topology.edges.$edgeId
        
        $edgePayload = @{
            id = $edgeId
            source = $edge.source
            target = $edge.target
            type = $edge.type
            metadata = @{
                source = "bridge"
                original_metadata = $edge.metadata
            }
        }
        
        if ($edge.protocol) {
            $edgePayload.metadata.protocol = $edge.protocol
        }
        
        if ($edge.connections) {
            $edgePayload.metadata.connections = $edge.connections
        }
        
        try {
            $jsonPayload = $edgePayload | ConvertTo-Json -Depth 3
            $response = Invoke-WebRequest -Uri "$topologyManagerURL/api/v1/topology/edges" -Method POST -ContentType "application/json" -Body $jsonPayload -TimeoutSec 10
            if ($response.StatusCode -eq 200 -or $response.StatusCode -eq 201) {
                Write-Host "Pushed edge: $edgeId" -ForegroundColor Green
            }
        } catch {
            Write-Host "Failed to push edge $edgeId`: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
}

# Main loop
while ($true) {
    Write-Host "Polling app server for topology data..." -ForegroundColor Blue
    
    $topologyData = Get-AppServerTopology
    if ($topologyData) {
        Write-Host "Found topology data: $($topologyData.topology.nodes.Count) nodes, $($topologyData.topology.edges.Count) edges" -ForegroundColor Green
        Push-ToTopologyManager $topologyData
    } else {
        Write-Host "No topology data available from app server" -ForegroundColor Yellow
    }
    
    Start-Sleep -Seconds $pollInterval
}
