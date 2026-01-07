package main

import (
	"log"

	"deployment-platform/internal/config"
	"deployment-platform/internal/database"
	"deployment-platform/internal/handlers/deployer"
	"deployment-platform/internal/handlers/user"
	"deployment-platform/internal/middleware"
	"deployment-platform/internal/services"
	deployerService "deployment-platform/internal/services/deployer"
	userService "deployment-platform/internal/services/user"

	"github.com/gin-gonic/gin"

	wsHandler "deployment-platform/internal/handlers/websocket"
	"deployment-platform/internal/services/websocket"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Connect to database
	db := database.Connect(cfg)

	// Auto Migrate
	database.AutoMigrate(db)

	// Initialize infrastructure services
	s3Service := services.NewS3Service(cfg)

	rabbitMQ, err := services.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitMQ.Close()

	// Initialize domain services

	// Initialize WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// DeployService needs: db, rmq, s3, hub
	deployServiceCore := services.NewDeployService(db, rabbitMQ, s3Service, hub)

	usrService := userService.NewService(db)
	depService := deployerService.NewService(db, deployServiceCore, cfg.BaseDomain)

	// Initialize handlers
	userHandler := user.NewHandler(usrService)
	deployHandler := deployer.NewHandler(depService)
	websocketHandler := wsHandler.NewHandler(hub)

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())

	// Routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/login", userHandler.Login)
	}

	api := r.Group("/")
	api.Use(middleware.Auth())
	{
		api.POST("/deploy", deployHandler.Deploy)
		api.GET("/deployments", deployHandler.GetDeployments)
		api.GET("/deployments/:id", deployHandler.GetStatus)
		api.DELETE("/deployments/:id", deployHandler.DeleteDeployment)
		api.GET("/deployments/:id/logs", websocketHandler.HandleLogs)
	}

	log.Printf("Server starting on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
