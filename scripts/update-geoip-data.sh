#!/bin/bash

# Update script for MaxMind GeoLite2 databases using geoipupdate
# This script updates existing databases or downloads them if they don't exist

set -e # Exit on error

# Configuration
MAXMIND_LICENSE_KEY="${MAXMIND_LICENSE_KEY:-}"
MAXMIND_ACCOUNT_ID="${MAXMIND_ACCOUNT_ID:-}"
DATA_DIR="${DATA_DIR:-./data}"
CONFIG_FILE="${DATA_DIR}/geoipupdate.conf"

echo "GeoIP Database Update Script"
echo "============================"

# Check required environment variables
if [ -z "${MAXMIND_LICENSE_KEY}" ]; then
    echo "Error: MAXMIND_LICENSE_KEY environment variable is required"
    exit 1
fi

if [ -z "${MAXMIND_ACCOUNT_ID}" ]; then
    echo "Warning: MAXMIND_ACCOUNT_ID not set, using default account ID"
    MAXMIND_ACCOUNT_ID="0"  # Default for GeoLite2
fi

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

# Generate geoipupdate configuration file
echo "Generating geoipupdate configuration..."
sed -e "s|__ACCOUNT_ID__|${MAXMIND_ACCOUNT_ID}|g" \
    -e "s|__LICENSE_KEY__|${MAXMIND_LICENSE_KEY}|g" \
    -e "s|__DATA_DIR__|${DATA_DIR}|g" \
    /app/scripts/geoipupdate.conf.template > "${CONFIG_FILE}"

echo "Configuration file created at: ${CONFIG_FILE}"

# Check if databases already exist
existing_dbs=$(find "${DATA_DIR}" -name "*.mmdb" -type f | wc -l)
if [ "$existing_dbs" -gt 0 ]; then
    echo "Found $existing_dbs existing database(s), updating..."
    action="Updating"
else
    echo "No existing databases found, downloading..."
    action="Downloading"
fi

# Run geoipupdate
echo "$action GeoLite2 databases..."
if geoipupdate -f "${CONFIG_FILE}" -v; then
    echo "✅ $action completed successfully!"
    
    # List downloaded files
    echo ""
    echo "Available database files:"
    ls -lh "${DATA_DIR}"/*.mmdb 2>/dev/null || echo "No .mmdb files found"
    
    # Show file dates to confirm updates
    echo ""
    echo "Database file timestamps:"
    find "${DATA_DIR}" -name "*.mmdb" -exec stat -c "%y %n" {} \;
else
    echo "❌ $action failed!"
    exit 1
fi

echo ""
echo "Update process completed at $(date)"
