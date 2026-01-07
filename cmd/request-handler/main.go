package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"deployment-platform/internal/config"
	"deployment-platform/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize services
	s3Service := services.NewS3Service(cfg)
	redisService := services.NewRedisService(cfg.RedisURL)

	r := gin.Default()

	r.GET("/*path", func(c *gin.Context) {
		host := c.Request.Host
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

		// Cache Key
		cacheKey := fmt.Sprintf("deploy:%s:%s", deployID, filePath)
		ctx := c.Request.Context()

		// 1. Check Redis Cache
		cachedContent, err := redisService.Get(ctx, cacheKey)
		if err == nil {
			contentType, _ := redisService.GetContentType(ctx, cacheKey)
			if contentType != "" {
				c.Header("Content-Type", contentType)
			}
			c.Writer.Write(cachedContent)
			return
		}

		// 2. Cache Miss - Fetch from S3
		key := fmt.Sprintf("dist/%s%s", deployID, filePath)
		output, err := s3Service.GetObject(key)
		if err != nil {
			// Try fallback to index.html for SPA
			if !strings.HasSuffix(filePath, "index.html") {
				fallbackKey := fmt.Sprintf("dist/%s/index.html", deployID)
				output, err = s3Service.GetObject(fallbackKey)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
					return
				}
				// Update cache key for fallback logic if we wanted to cache 404s/redirects,
				// but for now let's just serve the file.
				// To cache SPA index properly, we might want to cache it under the original key too,
				// but let's stick to simple caching for now.
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
				return
			}
		}
		defer output.Body.Close()

		// Read content
		content, err := io.ReadAll(output.Body)
		if err != nil {
			log.Printf("Error reading S3 content: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		// Set Content-Type
		contentType := ""
		if output.ContentType != nil {
			contentType = *output.ContentType
			c.Header("Content-Type", contentType)
		}

		// 3. Store in Redis (Async)
		go func() {
			bgCtx := context.Background()
			// Cache for 10 minutes
			if err := redisService.Set(bgCtx, cacheKey, content, 10*time.Minute); err != nil {
				log.Printf("Failed to cache content: %v", err)
			}
			if contentType != "" {
				redisService.SetContentType(bgCtx, cacheKey, contentType, 10*time.Minute)
			}
		}()

		// Stream content
		c.Writer.Write(content)
	})

	log.Printf("Request handler starting on port 3001")
	if err := r.Run(":3001"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
