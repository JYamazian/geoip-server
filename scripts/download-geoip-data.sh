#!/bin/bash

# Download script for MaxMind GeoLite2 databases
# This script downloads the GeoLite2 databases from MaxMind and extracts the .mmdb files

set -e # Exit on error

# Configuration
MAXMIND_LICENSE_KEY="${MAXMIND_LICENSE_KEY:-}"
DATA_DIR="${DATA_DIR:-./data}"

# Create data directory if it doesn't exist
mkdir -p "${DATA_DIR}"

echo "Downloading MaxMind GeoLite2 databases..."

if [ -z "${MAXMIND_LICENSE_KEY}" ]; then
    echo "Error: MAXMIND_LICENSE_KEY environment variable is required"
    echo "Please sign up for a free MaxMind account at https://www.maxmind.com/en/geolite2/signup"
    echo "and set your license key as an environment variable."
    exit 1
fi

# Database URLs
ASN_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
CITY_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
COUNTRY_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"

cd "${DATA_DIR}"

# Download and extract ASN database
echo "Downloading GeoLite2-ASN database..."
curl -f -L -o "GeoLite2-ASN.tar.gz" "${ASN_URL}"
echo "Extracting GeoLite2-ASN database..."
tar -xzf "GeoLite2-ASN.tar.gz"
find . -name "GeoLite2-ASN.mmdb" -exec mv {} . \;
rm -rf GeoLite2-ASN_* GeoLite2-ASN.tar.gz

# Download and extract City database
echo "Downloading GeoLite2-City database..."
curl -f -L -o "GeoLite2-City.tar.gz" "${CITY_URL}"
echo "Extracting GeoLite2-City database..."
tar -xzf "GeoLite2-City.tar.gz"
find . -name "GeoLite2-City.mmdb" -exec mv {} . \;
rm -rf GeoLite2-City_* GeoLite2-City.tar.gz

# Download and extract Country database
echo "Downloading GeoLite2-Country database..."
curl -f -L -o "GeoLite2-Country.tar.gz" "${COUNTRY_URL}"
echo "Extracting GeoLite2-Country database..."
tar -xzf "GeoLite2-Country.tar.gz"
find . -name "GeoLite2-Country.mmdb" -exec mv {} . \;
rm -rf GeoLite2-Country_* GeoLite2-Country.tar.gz

# Verify all files were downloaded successfully
if [ ! -f "${DATA_DIR}/GeoLite2-ASN.mmdb" ] || [ ! -f "${DATA_DIR}/GeoLite2-City.mmdb" ] || [ ! -f "${DATA_DIR}/GeoLite2-Country.mmdb" ]; then
    echo "One or more files are missing after download"
    exit 1
fi

echo "All databases downloaded successfully!"
echo "Database files:"
ls -lh "${DATA_DIR}"/*.mmdb
