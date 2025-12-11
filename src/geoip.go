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

// Whois returns GeoIP information as headers only, no response body
func (g *GeoIPService) Whois(c *gin.Context) {
	clientIP := getClientIP(c)

	ip := net.ParseIP(clientIP)
	if ip != nil {
		if cityRecord, err := g.cityDB.City(ip); err == nil {
			c.Header("X-GeoIP-Country", cityRecord.Country.IsoCode)
			c.Header("X-GeoIP-Country-Name", cityRecord.Country.Names["en"])
			c.Header("X-GeoIP-City", cityRecord.City.Names["en"])
			c.Header("X-GeoIP-Postal", cityRecord.Postal.Code)
			c.Header("X-GeoIP-Timezone", cityRecord.Location.TimeZone)
			if len(cityRecord.Subdivisions) > 0 {
				c.Header("X-GeoIP-Region", cityRecord.Subdivisions[0].Names["en"])
				c.Header("X-GeoIP-Region-Code", cityRecord.Subdivisions[0].IsoCode)
			}
		}
		if asnRecord, err := g.asnDB.ASN(ip); err == nil {
			c.Header("X-GeoIP-ASN", fmt.Sprintf("%d", asnRecord.AutonomousSystemNumber))
			c.Header("X-GeoIP-Organization", asnRecord.AutonomousSystemOrganization)
		}
	}
	c.Header("X-Client-IP", clientIP)
	c.Status(http.StatusNoContent)
}

// getClientIP extracts the client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check Cloudflare headers first - CF-Connecting-IP contains the original client IP
	cfConnectingIP := c.GetHeader("CF-Connecting-IP")
	if cfConnectingIP != "" && isValidPublicIP(cfConnectingIP) {
		return cfConnectingIP
	}

	// Check CF-IPCountry header to verify we're behind Cloudflare
	cfIPCountry := c.GetHeader("CF-IPCountry")
	if cfIPCountry != "" {
		// We're behind Cloudflare but CF-Connecting-IP wasn't found or valid
		// This shouldn't normally happen, but let's continue with other headers
	}

	// Check True-Client-IP (used by some CDNs and load balancers)
	trueClientIP := c.GetHeader("True-Client-IP")
	if trueClientIP != "" && isValidPublicIP(trueClientIP) {
		return trueClientIP
	}

	// Check X-Real-IP header (commonly used by nginx and other proxies)
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" && isValidPublicIP(xRealIP) {
		return xRealIP
	}

	// Check X-Forwarded-For header - this should contain the original client IP
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, check each one
		ips := strings.Split(xForwardedFor, ",")
		for _, ip := range ips {
			cleanIP := strings.TrimSpace(ip)
			// Return the first valid public IP
			if isValidPublicIP(cleanIP) {
				return cleanIP
			}
		}
		// If no public IP found, return the first IP anyway (might be internal but still useful)
		if len(ips) > 0 {
			firstIP := strings.TrimSpace(ips[0])
			if isValidIP(firstIP) {
				return firstIP
			}
		}
	}

	// Check X-Forwarded header
	xForwarded := c.GetHeader("X-Forwarded")
	if xForwarded != "" && isValidPublicIP(xForwarded) {
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
				// Handle IPv6 brackets
				forValue = strings.Trim(forValue, "[]")
				if isValidPublicIP(forValue) {
					return forValue
				}
			}
		}
	}

	// Fall back to RemoteAddr (this will likely be the pod IP in Kubernetes)
	return c.ClientIP()
}

// isPrivateIP checks if an IP address is private/internal
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",    // loopback
		"169.254.0.0/16", // link-local
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// isValidIP checks if a string is a valid IP address
func isValidIP(ipStr string) bool {
	return net.ParseIP(ipStr) != nil
}

// isValidPublicIP checks if an IP address is valid and public (not private)
func isValidPublicIP(ipStr string) bool {
	return isValidIP(ipStr) && !isPrivateIP(ipStr)
}
