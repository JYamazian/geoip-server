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

// GeoIPService handles GeoIP lookups using MaxMind databases
type GeoIPService struct {
	cityDB   *geoip2.Reader
	asnDB    *geoip2.Reader
	asnRawDB *maxminddb.Reader
}

// GeoIPResponse represents the response structure for GeoIP lookups
type GeoIPResponse struct {
	IP             string  `json:"ip"`
	Country        string  `json:"country"`
	CountryCode    string  `json:"country_code"`
	Region         string  `json:"region"`
	RegionCode     string  `json:"region_code"`
	City           string  `json:"city"`
	PostalCode     string  `json:"postal_code"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyRadius uint16  `json:"accuracy_radius,omitempty"`
	TimeZone       string  `json:"timezone"`
	ASN            uint    `json:"asn,omitempty"`
	ASNOrg         string  `json:"asn_org,omitempty"`
	ASNNetwork     string  `json:"asn_network,omitempty"`
}

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

	// Get ASN information
	asnRecord, asnErr := g.asnDB.ASN(ip)

	response := GeoIPResponse{
		IP:             ipStr,
		Country:        cityRecord.Country.Names["en"],
		CountryCode:    cityRecord.Country.IsoCode,
		City:           cityRecord.City.Names["en"],
		PostalCode:     cityRecord.Postal.Code,
		Latitude:       cityRecord.Location.Latitude,
		Longitude:      cityRecord.Location.Longitude,
		AccuracyRadius: cityRecord.Location.AccuracyRadius,
		TimeZone:       cityRecord.Location.TimeZone,
	}

	// Add ASN information if available
	if asnErr == nil {
		response.ASN = asnRecord.AutonomousSystemNumber
		response.ASNOrg = asnRecord.AutonomousSystemOrganization

		// Get ASN network information using the underlying maxminddb reader
		if g.asnRawDB != nil {
			var asnData map[string]interface{}
			if network, ok, err := g.asnRawDB.LookupNetwork(ip, &asnData); err == nil && ok && network != nil {
				response.ASNNetwork = network.String()
				fmt.Printf("DEBUG: ASN Network for %s: %s\n", ipStr, network.String())
			} else {
				fmt.Printf("DEBUG: ASN Network lookup failed for %s: err=%v, ok=%v, network=%v\n", ipStr, err, ok, network)
			}
		} else {
			fmt.Printf("DEBUG: asnRawDB is nil\n")
		}
	} else {
		fmt.Printf("DEBUG: ASN lookup failed for %s: %v\n", ipStr, asnErr)
	}

	// Add region information if available
	if len(cityRecord.Subdivisions) > 0 {
		response.Region = cityRecord.Subdivisions[0].Names["en"]
		response.RegionCode = cityRecord.Subdivisions[0].IsoCode
	}

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

	// Get ASN information
	asnRecord, asnErr := g.asnDB.ASN(ip)

	response := GeoIPResponse{
		IP:             clientIP,
		Country:        cityRecord.Country.Names["en"],
		CountryCode:    cityRecord.Country.IsoCode,
		City:           cityRecord.City.Names["en"],
		PostalCode:     cityRecord.Postal.Code,
		Latitude:       cityRecord.Location.Latitude,
		Longitude:      cityRecord.Location.Longitude,
		AccuracyRadius: cityRecord.Location.AccuracyRadius,
		TimeZone:       cityRecord.Location.TimeZone,
	}

	// Add ASN information if available
	if asnErr == nil {
		response.ASN = asnRecord.AutonomousSystemNumber
		response.ASNOrg = asnRecord.AutonomousSystemOrganization

		// Get ASN network information using the underlying maxminddb reader
		if g.asnRawDB != nil {
			var asnData map[string]interface{}
			if network, ok, err := g.asnRawDB.LookupNetwork(ip, &asnData); err == nil && ok && network != nil {
				response.ASNNetwork = network.String()
				fmt.Printf("DEBUG: Client ASN Network for %s: %s\n", clientIP, network.String())
			} else {
				fmt.Printf("DEBUG: Client ASN Network lookup failed for %s: err=%v, ok=%v, network=%v\n", clientIP, err, ok, network)
			}
		} else {
			fmt.Printf("DEBUG: asnRawDB is nil for client IP\n")
		}
	} else {
		fmt.Printf("DEBUG: Client ASN lookup failed for %s: %v\n", clientIP, asnErr)
	}

	// Add region information if available
	if len(cityRecord.Subdivisions) > 0 {
		response.Region = cityRecord.Subdivisions[0].Names["en"]
		response.RegionCode = cityRecord.Subdivisions[0].IsoCode
	}

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
