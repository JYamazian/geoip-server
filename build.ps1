# PowerShell script to build and run the GeoIP server with init container

Write-Host "Building GeoIP server images..." -ForegroundColor Green

try {
    # Build the init container image
    Write-Host "Building init container..." -ForegroundColor Yellow
    docker build -f Dockerfile.init -t geoip-init:latest .
    
    # Build the main application image  
    Write-Host "Building main application..." -ForegroundColor Yellow
    docker build -t geoip-server:latest .
    
    Write-Host "Images built successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "To run with Docker Compose:" -ForegroundColor Cyan
    Write-Host "  `$env:MAXMIND_LICENSE_KEY = 'your_license_key'" -ForegroundColor White
    Write-Host "  docker-compose up" -ForegroundColor White
    Write-Host ""
    Write-Host "To run manually:" -ForegroundColor Cyan
    Write-Host "  # First run the init container to download data:" -ForegroundColor White
    Write-Host "  docker run --rm -v geoip_data:/shared/data -e MAXMIND_LICENSE_KEY=your_key geoip-init:latest" -ForegroundColor White
    Write-Host "  # Then run the main server:" -ForegroundColor White
    Write-Host "  docker run -p 8080:8080 -v geoip_data:/shared/data -e DATA_DIR=/shared/data geoip-server:latest" -ForegroundColor White
}
catch {
    Write-Error "Build failed: $($_.Exception.Message)"
    exit 1
}
