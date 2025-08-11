#!/bin/bash

# Script to build and run the GeoIP server with init container

set -e

echo "Building GeoIP server images..."

# Build the init container image
echo "Building init container..."
docker build -f Dockerfile.init -t geoip-init:latest .

# Build the main application image  
echo "Building main application..."
docker build -t geoip-server:latest .

echo "Images built successfully!"
echo ""
echo "To run with Docker Compose:"
echo "  export MAXMIND_LICENSE_KEY=your_license_key"
echo "  docker-compose up"
echo ""
echo "To run manually:"
echo "  # First run the init container to download data:"
echo "  docker run --rm -v geoip_data:/shared/data -e MAXMIND_LICENSE_KEY=your_key geoip-init:latest"
echo "  # Then run the main server:"
echo "  docker run -p 8080:8080 -v geoip_data:/shared/data -e DATA_DIR=/shared/data geoip-server:latest"
