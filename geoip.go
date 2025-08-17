package main

import (
	"fmt"
	"log"
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
	// Log all headers for debugging
	log.Println("=== Client IP Debug Info ===")
	log.Printf("RemoteAddr: %s", c.Request.RemoteAddr)
	log.Printf("CF-Connecting-IP: %s", c.GetHeader("CF-Connecting-IP"))
	log.Printf("True-Client-IP: %s", c.GetHeader("True-Client-IP"))
	log.Printf("X-Real-IP: %s", c.GetHeader("X-Real-IP"))
	log.Printf("X-Forwarded-For: %s", c.GetHeader("X-Forwarded-For"))
	log.Printf("X-Forwarded: %s", c.GetHeader("X-Forwarded"))
	log.Printf("Forwarded: %s", c.GetHeader("Forwarded"))
	log.Printf("X-Client-IP: %s", c.GetHeader("X-Client-IP"))
	log.Printf("X-Cluster-Client-IP: %s", c.GetHeader("X-Cluster-Client-IP"))
	log.Printf("X-Original-Forwarded-For: %s", c.GetHeader("X-Original-Forwarded-For"))
	log.Printf("CF-IPCountry: %s", c.GetHeader("CF-IPCountry"))
	log.Printf("Gin ClientIP(): %s", c.ClientIP())
	
	// Log all headers for complete debugging
	log.Println("All headers:")
	for name, values := range c.Request.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	clientIP := getClientIP(c)
	log.Printf("Final extracted client IP: %s", clientIP)
	log.Println("=== End Debug Info ===")

	ip := net.ParseIP(clientIP)
	if ip == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unable to determine client IP",
			"debug": gin.H{
				"extracted_ip": clientIP,
				"remote_addr": c.Request.RemoteAddr,
				"headers": gin.H{
					"cf_connecting_ip": c.GetHeader("CF-Connecting-IP"),
					"x_real_ip": c.GetHeader("X-Real-IP"),
					"x_forwarded_for": c.GetHeader("X-Forwarded-For"),
				},
			},
		})
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

	// Add debug information to response
	response.Debug = gin.H{
		"remote_addr": c.Request.RemoteAddr,
		"headers": gin.H{
			"cf_connecting_ip": c.GetHeader("CF-Connecting-IP"),
			"x_real_ip": c.GetHeader("X-Real-IP"),
			"x_forwarded_for": c.GetHeader("X-Forwarded-For"),
			"gin_client_ip": c.ClientIP(),
		},
	}

	c.JSON(http.StatusOK, response)
}

// getClientIP extracts the client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check Cloudflare headers first - CF-Connecting-IP contains the original client IP
	cfConnectingIP := c.GetHeader("CF-Connecting-IP")
	if cfConnectingIP != "" && isValidPublicIP(cfConnectingIP) {
		log.Printf("Found valid CF-Connecting-IP: %s", cfConnectingIP)
		return cfConnectingIP
	}

	// Check CF-IPCountry header to verify we're behind Cloudflare
	cfIPCountry := c.GetHeader("CF-IPCountry")
	if cfIPCountry != "" {
		log.Printf("Behind Cloudflare (CF-IPCountry: %s) but CF-Connecting-IP not found or invalid: %s", cfIPCountry, cfConnectingIP)
	}

	// Check True-Client-IP (used by some CDNs and load balancers)
	trueClientIP := c.GetHeader("True-Client-IP")
	if trueClientIP != "" && isValidPublicIP(trueClientIP) {
		log.Printf("Found valid True-Client-IP: %s", trueClientIP)
		return trueClientIP
	}

	// Check X-Real-IP header (commonly used by nginx and other proxies)
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" && isValidPublicIP(xRealIP) {
		log.Printf("Found valid X-Real-IP: %s", xRealIP)
		return xRealIP
	}

	// Check X-Forwarded-For header - this should contain the original client IP
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		log.Printf("Processing X-Forwarded-For: %s", xForwardedFor)
		// X-Forwarded-For can contain multiple IPs, check each one
		ips := strings.Split(xForwardedFor, ",")
		for i, ip := range ips {
			cleanIP := strings.TrimSpace(ip)
			log.Printf("  IP %d: %s (valid: %t, public: %t)", i, cleanIP, isValidIP(cleanIP), isValidPublicIP(cleanIP))
			// Return the first valid public IP
			if isValidPublicIP(cleanIP) {
				log.Printf("Selected public IP from X-Forwarded-For: %s", cleanIP)
				return cleanIP
			}
		}
		// If no public IP found, return the first valid IP anyway (might be internal but still useful)
		if len(ips) > 0 {
			firstIP := strings.TrimSpace(ips[0])
			if isValidIP(firstIP) {
				log.Printf("No public IP found, using first IP from X-Forwarded-For: %s", firstIP)
				return firstIP
			}
		}
	}

	// Check additional headers commonly used by various proxies and load balancers
	headers := []string{
		"X-Client-IP",
		"X-Cluster-Client-IP", 
		"X-Original-Forwarded-For",
		"X-Forwarded",
	}
	
	for _, header := range headers {
		value := c.GetHeader(header)
		if value != "" && isValidPublicIP(value) {
			log.Printf("Found valid %s: %s", header, value)
			return value
		}
	}

	// Check Forwarded header (RFC 7239)
	forwarded := c.GetHeader("Forwarded")
	if forwarded != "" {
		log.Printf("Processing Forwarded header: %s", forwarded)
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
					log.Printf("Found valid IP in Forwarded header: %s", forValue)
					return forValue
				}
			}
		}
	}

	// Fall back to RemoteAddr (this will likely be the pod IP in Kubernetes)
	remoteAddr := c.ClientIP()
	log.Printf("Falling back to RemoteAddr: %s", remoteAddr)
	
	// If RemoteAddr is a private IP (like 172.18.0.1), it means no proxy headers were set
	if isPrivateIP(remoteAddr) {
		log.Printf("WARNING: Returning private IP %s - no valid proxy headers found. Check proxy configuration.", remoteAddr)
	}
	
	return remoteAddr
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

// ForwardAuthLookup handles ForwardAuth requests from Traefik
// This endpoint is specifically designed for Traefik ForwardAuth middleware
func (g *GeoIPService) ForwardAuthLookup(c *gin.Context) {
	// Get the client IP from headers set by Traefik
	clientIP := getClientIP(c)

	ip := net.ParseIP(clientIP)
	if ip == nil {
		// Return 200 but without geo headers if IP can't be parsed
		c.Status(http.StatusOK)
		return
	}

	cityRecord, err := g.cityDB.City(ip)
	if err != nil {
		// Return 200 but without geo headers if lookup fails
		c.Status(http.StatusOK)
		return
	}

	// Create base response using helper function to get all the data
	response := NewGeoIPResponse(clientIP, cityRecord)

	// Add ASN information using helper function
	AddASNInformation(&response, ip, clientIP, g.asnDB, g.asnRawDB)

	// Set all geographic and ASN information as headers for ForwardAuth
	c.Header("X-GeoIP-IP", response.IP)
	c.Header("X-GeoIP-Country", response.CountryCode)
	c.Header("X-GeoIP-Country-Name", response.Country)
	c.Header("X-GeoIP-Region", response.RegionCode)
	c.Header("X-GeoIP-Region-Name", response.Region)
	c.Header("X-GeoIP-City", response.City)
	c.Header("X-GeoIP-Postal-Code", response.PostalCode)
	c.Header("X-GeoIP-Latitude", fmt.Sprintf("%.6f", response.Latitude))
	c.Header("X-GeoIP-Longitude", fmt.Sprintf("%.6f", response.Longitude))
	c.Header("X-GeoIP-Accuracy-Radius", fmt.Sprintf("%d", response.AccuracyRadius))
	c.Header("X-GeoIP-Timezone", response.TimeZone)

	// Add ASN headers if available
	if response.ASN != 0 {
		c.Header("X-GeoIP-ASN", fmt.Sprintf("%d", response.ASN))
	}
	if response.ASNOrg != "" {
		c.Header("X-GeoIP-ASN-Org", response.ASNOrg)
	}
	if response.ASNNetwork != "" {
		c.Header("X-GeoIP-ASN-Network", response.ASNNetwork)
	}

	// Forward the client IP for downstream services
	c.Header("X-Forwarded-For", clientIP)

	// Return 200 OK to allow the request to proceed
	c.Status(http.StatusOK)
}
