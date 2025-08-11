# Manual update trigger script (PowerShell)
# This script runs a one-time update of the GeoIP databases

Write-Host "Triggering manual GeoIP database update..." -ForegroundColor Green

# Check if Docker Compose is available
$composeCmd = $null
if (Get-Command docker-compose -ErrorAction SilentlyContinue) {
    $composeCmd = "docker-compose"
} elseif (Get-Command docker -ErrorAction SilentlyContinue) {
    try {
        docker compose version | Out-Null
        $composeCmd = "docker compose"
    } catch {
        # Docker compose plugin not available
    }
}

if (-not $composeCmd) {
    Write-Error "Docker Compose not found"
    exit 1
}

# Run a one-time update
Write-Host "Running database update..." -ForegroundColor Yellow
& $composeCmd run --rm -e MODE=update geoip-data-downloader

Write-Host "Update completed!" -ForegroundColor Green
Write-Host ""
Write-Host "You may need to restart the geoip-server service to use the updated databases:" -ForegroundColor Cyan
Write-Host "  $composeCmd restart geoip-server" -ForegroundColor White
