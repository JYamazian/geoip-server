#!/bin/bash

# Manual update trigger script
# This script runs a one-time update of the GeoIP databases

echo "Triggering manual GeoIP database update..."

# Check if Docker Compose is available
if command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    echo "Error: Docker Compose not found"
    exit 1
fi

# Run a one-time update
echo "Running database update..."
$COMPOSE_CMD run --rm \
    -e MODE=update \
    geoip-data-downloader

echo "Update completed!"
echo ""
echo "You may need to restart the geoip-server service to use the updated databases:"
echo "  $COMPOSE_CMD restart geoip-server"
