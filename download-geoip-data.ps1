# PowerShell script to download MaxMind GeoLite2 databases
# This script downloads the GeoLite2 databases from MaxMind and extracts the .mmdb files

param(
    [string]$LicenseKey = $env:MAXMIND_LICENSE_KEY,
    [string]$DataDir = ".\data"
)

# Check if license key is provided
if (-not $LicenseKey) {
    Write-Error "MAXMIND_LICENSE_KEY is required. Please set the environment variable or pass it as a parameter."
    Write-Host "Sign up for a free MaxMind account at https://www.maxmind.com/en/geolite2/signup"
    exit 1
}

# Create data directory if it doesn't exist
if (-not (Test-Path $DataDir)) {
    New-Item -ItemType Directory -Path $DataDir -Force | Out-Null
}

# Configuration
$DatabaseUrls = @{
    "GeoLite2-ASN" = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=$LicenseKey&suffix=tar.gz"
    "GeoLite2-City" = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=$LicenseKey&suffix=tar.gz"
    "GeoLite2-Country" = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=$LicenseKey&suffix=tar.gz"
}

Write-Host "Downloading MaxMind GeoLite2 databases..." -ForegroundColor Green

try {
    foreach ($database in $DatabaseUrls.GetEnumerator()) {
        $dbName = $database.Key
        $url = $database.Value
        $tarGzFile = Join-Path $DataDir "$dbName.tar.gz"
        $mmdbFile = Join-Path $DataDir "$dbName.mmdb"
        
        Write-Host "Downloading $dbName..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $url -OutFile $tarGzFile -UseBasicParsing
        
        Write-Host "Extracting $dbName..." -ForegroundColor Yellow
        
        # Extract using tar (available in Windows 10 1903+ and Windows Server 2019+)
        if (Get-Command tar -ErrorAction SilentlyContinue) {
            Push-Location $DataDir
            tar -xzf "$dbName.tar.gz"
            
            # Find and move the .mmdb file
            $mmdbFiles = Get-ChildItem -Recurse -Filter "$dbName.mmdb"
            if ($mmdbFiles.Count -gt 0) {
                Move-Item $mmdbFiles[0].FullName -Destination $mmdbFile -Force
            }
            
            # Clean up extracted directories
            Get-ChildItem -Directory -Filter "${dbName}_*" | Remove-Item -Recurse -Force
            Remove-Item "$dbName.tar.gz" -Force
            Pop-Location
        } else {
            Write-Error "tar command not found. Please install tar or use WSL/Git Bash."
            exit 1
        }
        
        if (-not (Test-Path $mmdbFile)) {
            throw "Failed to extract $dbName.mmdb"
        }
    }
    
    # Verify all files were downloaded
    $downloadedFiles = Get-ChildItem -Path $DataDir -Filter "*.mmdb"
    if ($downloadedFiles.Count -ne 3) {
        throw "Not all database files were downloaded successfully"
    }
    
    Write-Host "All databases downloaded successfully!" -ForegroundColor Green
    Write-Host "Downloaded files:" -ForegroundColor Cyan
    $downloadedFiles | ForEach-Object {
        $size = [math]::Round($_.Length / 1MB, 2)
        Write-Host "  $($_.Name) - $size MB" -ForegroundColor White
    }
    
} catch {
    Write-Error "Failed to download databases: $($_.Exception.Message)"
    exit 1
}
