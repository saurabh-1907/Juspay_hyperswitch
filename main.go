package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/saurabh/payment-routing-layer/internal/db"
	"github.com/saurabh/payment-routing-layer/internal/handlers"
	"github.com/saurabh/payment-routing-layer/internal/middleware"
)

func main() {
	// Load environment variables if .env exists
	_ = godotenv.Load()

	// Initialize Database Connections
	db.ConnectMongo()
	defer db.DisconnectMongo()
	
	db.ConnectRedis()

	// Setup Router
	r := gin.Default()

	// Serve Static Files for Frontend
	r.StaticFS("/ui", http.Dir("static"))
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})

	// API Routes
	api := r.Group("/api")
	{
		api.POST("/payments", middleware.IdempotencyMiddleware(), handlers.CreatePayment)
		api.GET("/payments/:orderId/reconcile", handlers.ReconcilePayment)
		api.POST("/webhooks", handlers.HandleHyperswitchWebhook)
	}

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server is running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
