package main

import (
	"fmt"
	"net"

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

// NewGeoIPResponse creates a new GeoIPResponse from a city record and IP string
func NewGeoIPResponse(ipStr string, cityRecord *geoip2.City) GeoIPResponse {
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

	// Add region information if available
	if len(cityRecord.Subdivisions) > 0 {
		response.Region = cityRecord.Subdivisions[0].Names["en"]
		response.RegionCode = cityRecord.Subdivisions[0].IsoCode
	}

	return response
}

// AddASNInformation adds ASN information to a GeoIPResponse
func AddASNInformation(response *GeoIPResponse, ip net.IP, ipStr string, asnDB *geoip2.Reader, asnRawDB *maxminddb.Reader) {
	// Get ASN information
	asnRecord, asnErr := asnDB.ASN(ip)

	// Add ASN information if available
	if asnErr == nil {
		response.ASN = asnRecord.AutonomousSystemNumber
		response.ASNOrg = asnRecord.AutonomousSystemOrganization

		// Get ASN network information using the underlying maxminddb reader
		if asnRawDB != nil {
			var asnData map[string]interface{}
			if network, ok, err := asnRawDB.LookupNetwork(ip, &asnData); err == nil && ok && network != nil {
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
}
