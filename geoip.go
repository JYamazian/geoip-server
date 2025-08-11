package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/oschwald/geoip2-golang"
	"github.com/oschwald/maxminddb-golang"
)

// NewGeoIPService creates a new GeoIP service instance
func NewGeoIPService(dataDir string) (*GeoIPService, error) {
	// Open City database
	cityDB, err := geoip2.Open(dataDir + "/GeoLite2-City.mmdb")
	if err != nil {
		return nil, fmt.Errorf("failed to open City database: %w", err)
	}

	// Open ASN database
	asnDB, err := geoip2.Open(dataDir + "/GeoLite2-ASN.mmdb")
	if err != nil {
		cityDB.Close() // Clean up city DB if ASN fails
		return nil, fmt.Errorf("failed to open ASN database: %w", err)
	}

	// Open ASN database with maxminddb for network information
	asnRawDB, err := maxminddb.Open(dataDir + "/GeoLite2-ASN.mmdb")
	if err != nil {
		cityDB.Close()
		asnDB.Close()
		return nil, fmt.Errorf("failed to open ASN raw database: %w", err)
	}

	return &GeoIPService{
		cityDB:   cityDB,
		asnDB:    asnDB,
		asnRawDB: asnRawDB,
	}, nil
}

// Close closes the GeoIP databases
func (g *GeoIPService) Close() error {
	var err1, err2, err3 error
	if g.cityDB != nil {
		err1 = g.cityDB.Close()
	}
	if g.asnDB != nil {
		err2 = g.asnDB.Close()
	}
	if g.asnRawDB != nil {
		err3 = g.asnRawDB.Close()
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}

// LookupIP handles IP lookup requests
func (g *GeoIPService) LookupIP(c *gin.Context) {
	ipStr := c.Param("ip")
	if ipStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "IP address is required"})
		return
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IP address format"})
		return
	}

	cityRecord, err := g.cityDB.City(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lookup IP address"})
		return
	}

	// Create base response using helper function
	response := NewGeoIPResponse(ipStr, cityRecord)

	// Add ASN information using helper function
	AddASNInformation(&response, ip, ipStr, g.asnDB, g.asnRawDB)

	c.JSON(http.StatusOK, response)
}

// GetClientIP returns information about the client's IP address
func (g *GeoIPService) GetClientIP(c *gin.Context) {
	clientIP := getClientIP(c)

	ip := net.ParseIP(clientIP)
	if ip == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to determine client IP"})
		return
	}

	cityRecord, err := g.cityDB.City(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lookup client IP address"})
		return
	}

	// Create base response using helper function
	response := NewGeoIPResponse(clientIP, cityRecord)

	// Add ASN information using helper function
	AddASNInformation(&response, ip, clientIP, g.asnDB, g.asnRawDB)

	c.JSON(http.StatusOK, response)
}

// getClientIP extracts the client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// Check X-Forwarded header
	xForwarded := c.GetHeader("X-Forwarded")
	if xForwarded != "" {
		return xForwarded
	}

	// Check Forwarded header (RFC 7239)
	forwarded := c.GetHeader("Forwarded")
	if forwarded != "" {
		// Parse the Forwarded header for the "for" field
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			if strings.HasPrefix(strings.TrimSpace(part), "for=") {
				forValue := strings.TrimPrefix(strings.TrimSpace(part), "for=")
				// Remove quotes if present
				forValue = strings.Trim(forValue, "\"")
				return forValue
			}
		}
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}
