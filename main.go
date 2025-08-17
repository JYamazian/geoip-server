package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// getClientIPForLogging is a simple version for logging purposes
func getClientIPForLogging(c *gin.Context) string {
	// Check Cloudflare first
	if cfIP := c.GetHeader("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}
	// Check X-Real-IP
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}
	// Check X-Forwarded-For
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return c.ClientIP()
}

func main() {
	// Get data directory from environment or use default
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Initialize the GeoIP service with data directory
	geoIPService, err := NewGeoIPService(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize GeoIP service: %v", err)
	}
	defer geoIPService.Close()

	// Create Gin router
	r := gin.Default()

	// Configure Gin to trust all proxies (needed for Kubernetes and Cloudflare)
	// This allows Gin to properly parse proxy headers
	r.SetTrustedProxies(nil) // Trust all proxies
	
	// Add middleware to log client IP for debugging
	r.Use(func(c *gin.Context) {
		log.Printf("Request from IP: %s, CF-Connecting-IP: %s, X-Forwarded-For: %s, X-Real-IP: %s, RemoteAddr: %s",
			getClientIPForLogging(c),
			c.GetHeader("CF-Connecting-IP"),
			c.GetHeader("X-Forwarded-For"),
			c.GetHeader("X-Real-IP"),
			c.Request.RemoteAddr)
		c.Next()
	})

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		})
	})

	// GeoIP lookup endpoint
	r.GET("/:ip", geoIPService.LookupIP)

	// Get client IP info
	r.GET("/myip", geoIPService.GetClientIP)

	// Set up graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Starting GeoIP server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
