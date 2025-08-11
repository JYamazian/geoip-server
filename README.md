# GeoIP Server

A high-performance GeoIP server written in Go that provides location information based on IP addresses using MaxMind's GeoLite2 database.

## Features

- Fast IP geolocation lookups
- RESTful API endpoints
- Support for both IPv4 and IPv6
- Client IP detection with proper proxy header handling
- Health check endpoint
- Graceful shutdown
- CORS support
- Docker support with init container for data download

## Prerequisites

1. **MaxMind License Key**: You need a free MaxMind account and license key
   - Sign up at: https://www.maxmind.com/en/geolite2/signup
   - Get your license key from your account dashboard

## Quick Start

### Option 1: Local Development

1. **Set up your MaxMind license key:**
   ```bash
   export MAXMIND_LICENSE_KEY="your_license_key_here"
   ```

2. **Download the GeoIP database:**
   ```bash
   chmod +x download-geoip-data.sh
   ./download-geoip-data.sh
   ```

3. **Install dependencies:**
   ```bash
   go mod download
   ```

4. **Run the server:**
   ```bash
   go run .
   ```

### Option 2: Docker

1. **Build and run with Docker:**
   ```bash
   docker build -t geoip-server .
   docker run -p 8080:8080 -e MAXMIND_LICENSE_KEY="your_license_key_here" geoip-server
   ```

### Option 3: Docker Compose

1. **Create a `.env` file:**
   ```
   MAXMIND_LICENSE_KEY=your_license_key_here
   ```

2. **Run with Docker Compose:**
   ```bash
   docker-compose up
   ```

## API Endpoints

### Health Check
```
GET /health
```
Returns server health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2023-12-07T10:30:00Z"
}
```

### IP Lookup
```
GET /geoip/{ip_address}
```
Returns geolocation information for the specified IP address.

**Example:**
```bash
curl http://localhost:8080/geoip/8.8.8.8
```

**Response:**
```json
{
  "ip": "8.8.8.8",
  "country": "United States",
  "country_code": "US",
  "region": "California",
  "region_code": "CA",
  "city": "Mountain View",
  "postal_code": "94043",
  "latitude": 37.4056,
  "longitude": -122.0775,
  "timezone": "America/Los_Angeles"
}
```

### Client IP Information
```
GET /myip
```
Returns geolocation information for the client's IP address.

**Response:**
```json
{
  "ip": "203.0.113.1",
  "country": "Australia",
  "country_code": "AU",
  "region": "New South Wales",
  "region_code": "NSW",
  "city": "Sydney",
  "postal_code": "2000",
  "latitude": -33.8688,
  "longitude": 151.2093,
  "timezone": "Australia/Sydney"
}
```

## Database Management

The server includes flexible database management with both initial download and update capabilities:

### Initial Setup
The init container automatically downloads databases on first startup.

### Manual Updates
Update databases on-demand using the update scripts:

**Linux/macOS:**
```bash
chmod +x update-databases.sh
./update-databases.sh
```

**Windows (PowerShell):**
```powershell
.\update-databases.ps1
```

### Automatic Updates (Optional)
Enable continuous updates that check for new databases every 24 hours:

```bash
# Start with automatic updates enabled
docker-compose --profile updater up
```

### Environment Variables

- `MAXMIND_LICENSE_KEY`: Your MaxMind license key (required)
- `MAXMIND_ACCOUNT_ID`: Your MaxMind account ID (optional, defaults to 0 for GeoLite2)
- `DATA_DIR`: Directory where databases are stored (default: varies by deployment)
- `MODE`: Operation mode for database manager (`init`, `update`, `daemon`)

## Development

### Project Structure

```
.
├── main.go                 # Application entry point
├── geoip.go               # GeoIP service implementation
├── download-geoip-data.sh # Database download script
├── Dockerfile             # Docker configuration
├── docker-compose.yml     # Docker Compose configuration
├── go.mod                 # Go module dependencies
└── README.md              # This file
```

### Building

```bash
go build -o geoip-server .
```

### Testing

Test the endpoints:

```bash
# Health check
curl http://localhost:8080/health

# IP lookup
curl http://localhost:8080/geoip/8.8.8.8

# Client IP
curl http://localhost:8080/myip
```

## Deployment

### Kubernetes

The application is designed to work well in Kubernetes environments with init containers for database downloads. The Docker image can be used with an init container pattern where the init container downloads the MaxMind database before the main application starts.

### Production Considerations

1. **Database Updates**: The GeoLite2 database is updated regularly. Consider setting up a cron job or scheduled task to update the database periodically.

2. **Security**: 
   - Keep your MaxMind license key secure
   - Consider using secrets management for the license key in production
   - Use HTTPS in production environments

3. **Performance**: 
   - The application loads the entire database into memory for fast lookups
   - Monitor memory usage based on your database size
   - Consider horizontal scaling for high-traffic scenarios

4. **Monitoring**: 
   - Use the `/health` endpoint for health checks
   - Monitor response times and error rates
   - Set up logging aggregation for troubleshooting

## License

This project is open source. The MaxMind GeoLite2 database is licensed under the Creative Commons Attribution-ShareAlike 4.0 International License.

## Acknowledgments

- MaxMind for providing the GeoLite2 database
- The Go GeoIP2 library maintainers
- Gin web framework for the HTTP server
