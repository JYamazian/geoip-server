#!/bin/bash

# Combined init and update script for MaxMind GeoLite2 databases
# This script can download databases initially and also update them

set -e # Exit on error

# Configuration
MAXMIND_LICENSE_KEY="${MAXMIND_LICENSE_KEY:-}"
MAXMIND_ACCOUNT_ID="${MAXMIND_ACCOUNT_ID:-0}"  # Default for GeoLite2
DATA_DIR="${DATA_DIR:-./data}"
MODE="${MODE:-update}"  # Can be 'init', 'update', or 'daemon'
UPDATE_INTERVAL="${UPDATE_INTERVAL:-24h}"  # Interval for daemon mode, e.g. "24h" or "30m"

echo "GeoIP Database Manager"
echo "====================="
echo "Mode: $MODE"
echo "Data Directory: $DATA_DIR"
echo "Maxmind License Key: $MAXMIND_LICENSE_KEY"
echo ""

# Check required environment variables
if [ -z "${MAXMIND_LICENSE_KEY}" ]; then
    echo "Error: MAXMIND_LICENSE_KEY environment variable is required"
    exit 1
fi

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Function to download using direct API (fallback)
download_direct() {
    echo "Using direct download method..."
    
    # Database URLs
    ASN_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
    CITY_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
    COUNTRY_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"

    cd "${DATA_DIR}"

    # Download and extract ASN database
    echo "Downloading GeoLite2-ASN database..."
    curl -f -L -o "GeoLite2-ASN.tar.gz" "${ASN_URL}"
    tar -xzf "GeoLite2-ASN.tar.gz"
    find . -name "GeoLite2-ASN.mmdb" -exec mv {} . \;
    rm -rf GeoLite2-ASN_* GeoLite2-ASN.tar.gz

    # Download and extract City database
    echo "Downloading GeoLite2-City database..."
    curl -f -L -o "GeoLite2-City.tar.gz" "${CITY_URL}"
    tar -xzf "GeoLite2-City.tar.gz"
    find . -name "GeoLite2-City.mmdb" -exec mv {} . \;
    rm -rf GeoLite2-City_* GeoLite2-City.tar.gz

    # Download and extract Country database
    echo "Downloading GeoLite2-Country database..."
    curl -f -L -o "GeoLite2-Country.tar.gz" "${COUNTRY_URL}"
    tar -xzf "GeoLite2-Country.tar.gz"
    find . -name "GeoLite2-Country.mmdb" -exec mv {} . \;
    rm -rf GeoLite2-Country_* GeoLite2-Country.tar.gz
}

# Function to use geoipupdate
update_with_geoipupdate() {
    echo "Using geoipupdate for database management..."
    
    CONFIG_FILE="${DATA_DIR}/geoipupdate.conf"
    
    # Generate geoipupdate configuration file
    sed -e "s|__ACCOUNT_ID__|${MAXMIND_ACCOUNT_ID}|g" \
        -e "s|__LICENSE_KEY__|${MAXMIND_LICENSE_KEY}|g" \
        -e "s|__DATA_DIR__|${DATA_DIR}|g" \
        /app/geoipupdate.conf.template > "${CONFIG_FILE}"

    # Run geoipupdate
    geoipupdate -f "${CONFIG_FILE}" -v
}

# Main execution based on mode
case "$MODE" in
    "init")
        echo "Initializing databases (first-time download)..."
        existing_dbs=$(find "${DATA_DIR}" -name "*.mmdb" -type f 2>/dev/null | wc -l)
        
        if [ "$existing_dbs" -gt 0 ]; then
            echo "Databases already exist, skipping download"
        else
            # Try geoipupdate first, fallback to direct download
            if command -v geoipupdate >/dev/null 2>&1; then
                if ! update_with_geoipupdate; then
                    echo "geoipupdate failed, falling back to direct download..."
                    download_direct
                fi
            else
                echo "geoipupdate not available, using direct download..."
                download_direct
            fi
        fi
        ;;
        
    "update")
        echo "Updating existing databases..."
        if command -v geoipupdate >/dev/null 2>&1; then
            update_with_geoipupdate
        else
            echo "geoipupdate not available, using direct download..."
            download_direct
        fi
        ;;
        
    "daemon")
        echo "Starting daemon mode (multiple intervals supported)..."
        # Allow comma-separated intervals, e.g. "30s,5m,3h"
        IFS=',' read -ra INTERVALS <<< "${UPDATE_INTERVAL}"
        while true; do
            for interval in "${INTERVALS[@]}"; do
                echo "$(date): Updating databases..."
                if command -v geoipupdate >/dev/null 2>&1; then
                    update_with_geoipupdate || echo "Update failed, will retry in $interval"
                else
                    download_direct || echo "Download failed, will retry in $interval"
                fi
                echo "$(date): Next update in $interval"
                sleep "$interval"
            done
        done
        ;;
        
    *)
        echo "Invalid mode: $MODE"
        echo "Valid modes: init, update, daemon"
        exit 1
        ;;
esac

# Verify databases exist
echo ""
echo "Verifying databases..."
required_dbs=("GeoLite2-ASN.mmdb" "GeoLite2-City.mmdb" "GeoLite2-Country.mmdb")
missing_dbs=0

for db in "${required_dbs[@]}"; do
    if [ ! -f "${DATA_DIR}/${db}" ]; then
        echo "❌ Missing: ${db}"
        missing_dbs=$((missing_dbs + 1))
    else
        echo "✅ Found: ${db}"
    fi
done

if [ $missing_dbs -eq 0 ]; then
    echo ""
    echo "✅ All databases are available!"
    echo "Database files:"
    ls -lh "${DATA_DIR}"/*.mmdb
else
    echo ""
    echo "❌ $missing_dbs database(s) missing!"
    exit 1
fi

echo ""
echo "Database management completed successfully at $(date)"
