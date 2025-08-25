
# GeoIP Server

[![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Gin](https://img.shields.io/badge/Gin-Web_Framework-00ADD8?logo=go&logoColor=white)](https://gin-gonic.com/)
[![MaxMind](https://img.shields.io/badge/MaxMind-GeoLite2-FF6B35?logo=maxmind&logoColor=white)](https://www.maxmind.com/)

GeoIP Server is a high-performance Go service providing IP geolocation using MaxMind's GeoLite2 databases. It supports Docker, Kubernetes, and local development.

## Features

- Fast IP geolocation lookups (IPv4 & IPv6)
- RESTful API endpoints
- Smart client IP detection (proxy headers)
- Health check endpoint
- CORS support
- Docker & Kubernetes ready
- Automated database download/update

docker build -t geoip-server .
docker run -p 8080:8080 -e MAXMIND_LICENSE_KEY="your_license_key_here" geoip-server
## Quick Start

### Prerequisites

- Free MaxMind account & license key ([Sign up](https://www.maxmind.com/en/geolite2/signup))

### Docker (Recommended)

```sh
docker build -t geoip-server .
docker run -p 8080:8080 -e MAXMIND_LICENSE_KEY=your_license_key geoip-server
```

### Docker Compose

1. Create `.env` file:
   ```env
   MAXMIND_LICENSE_KEY=your_license_key
   ```
2. Edit `docker-compose.yml` to use env vars (not hardcoded keys)
3. Start services:
   ```sh
   docker-compose up -d --build
   ```

### Local Development

```sh
export MAXMIND_LICENSE_KEY=your_license_key
chmod +x scripts/download-geoip-data.sh
./scripts/download-geoip-data.sh
go mod download
go run src/
```

## API Reference

### Health Check
`GET /health`

Returns server health status:
```json
{
   "status": "healthy",
   "timestamp": "2023-12-07T10:30:00Z"
}
```

### IP Geolocation Lookup
`GET /{ip}`

Returns geolocation info for any IP:
```sh
curl http://localhost:8080/8.8.8.8
```
Response:
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
   "accuracy_radius": 1000,
   "timezone": "America/Los_Angeles",
   "asn": 15169,
   "asn_org": "Google LLC",
   "asn_network": "8.8.8.0/24"
}
```

### Client IP Info
`GET /myip`

Returns geolocation info for the requester's IP.

docker-compose --profile updater up -d
## Database Management

Uses **GeoLite2-City** and **GeoLite2-ASN** databases from MaxMind.

- Initial download: automatic in Docker/Kubernetes via init container
- Manual download: `scripts/download-geoip-data.sh`
- Update: `scripts/update-geoip-data.sh` or `scripts/geoip-manager.sh` (supports daemon mode with `UPDATE_INTERVAL`)

## Configuration

Environment variables:

- `MAXMIND_LICENSE_KEY` (required)
- `MAXMIND_ACCOUNT_ID` (optional, default: 0)
- `DATA_DIR` (optional, default: ./data)
- `MODE` (`init`, `update`, `daemon`)
- `UPDATE_INTERVAL` (for daemon mode, e.g. "30s,5m,3h")

go build -o geoip-server ./src
curl http://localhost:8080/8.8.8.8
## Development

Project structure:
```
geoip-server/
├── src/                       # Go source code
│   ├── main.go
│   ├── geoip.go
│   └── types.go
├── scripts/                   # Database management scripts
│   ├── download-geoip-data.sh
│   ├── update-geoip-data.sh
│   ├── geoip-manager.sh
│   └── geoipupdate.conf.template
├── Dockerfile                 # Main app image
├── Dockerfile.init            # Init container image
├── docker-compose.yml         # Compose config
├── go.mod, go.sum             # Go modules
├── Makefile                   # Build automation
└── README.md
```

Build & run locally:
```sh
go build -o geoip-server ./src
./geoip-server
```

Test endpoints:
```sh
curl http://localhost:8080/health
curl http://localhost:8080/8.8.8.8
curl http://localhost:8080/myip
```

## Deployment

### Docker
- Build main app: `docker build -t geoip-server .`
- Build init container: `docker build -f Dockerfile.init -t geoip-server-init .`
- Use both images in Kubernetes or Compose as needed

### Kubernetes Example
```yaml
initContainers:
   - name: geoip-init
      image: geoip-server-init:latest
      env:
         - name: MAXMIND_LICENSE_KEY
            valueFrom:
               secretKeyRef:
                  name: maxmind-secret
                  key: license-key
      volumeMounts:
         - name: geoip-data
            mountPath: /shared/data
containers:
   - name: geoip-server
      image: geoip-server:latest
      env:
         - name: DATA_DIR
            value: /shared/data
      volumeMounts:
         - name: geoip-data
            mountPath: /shared/data
volumes:
   - name: geoip-data
      emptyDir: {}
```

## Production Notes
- Keep your MaxMind license key secure (use secrets)
- Monitor database freshness and server health
- Use HTTPS and rate limiting in production
- Horizontal scaling recommended for high traffic

## License
MIT License

---

[Report Bug](https://github.com/JYamazian/geoip-server/issues) • [Request Feature](https://github.com/JYamazian/geoip-server/issues) • [Contribute](https://github.com/JYamazian/geoip-server/pulls)
