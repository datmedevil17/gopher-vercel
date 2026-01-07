package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"deployment-platform/internal/config"
	"deployment-platform/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize S3 service
	s3Service := services.NewS3Service(cfg)

	r := gin.Default()

	r.GET("/*path", func(c *gin.Context) {
		host := c.Request.Host
		// Extract subdomain: id.domain.com -> id
		// For localhost testing: id.localhost:3001 -> id
		parts := strings.Split(host, ".")
		if len(parts) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hostname"})
			return
		}
		deployID := parts[0]

		filePath := c.Param("path")
		if filePath == "/" || filePath == "" {
			filePath = "/index.html"
		}

		// Construct S3 key
		key := fmt.Sprintf("dist/%s%s", deployID, filePath)

		// Fetch from S3
		output, err := s3Service.GetObject(key)
		if err != nil {
			// If file not found, try serving index.html for SPA support
			if !strings.HasSuffix(filePath, "index.html") {
				fallbackKey := fmt.Sprintf("dist/%s/index.html", deployID)
				output, err = s3Service.GetObject(fallbackKey)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
					return
				}
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
				return
			}
		}
		defer output.Body.Close()

		// Set Content-Type
		if output.ContentType != nil {
			c.Header("Content-Type", *output.ContentType)
		}

		// Stream content
		_, err = io.Copy(c.Writer, output.Body)
		if err != nil {
			log.Printf("Error streaming file: %v", err)
		}
	})

	log.Printf("Request handler starting on port 3001")
	if err := r.Run(":3001"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
