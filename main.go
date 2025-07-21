package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type HTMLRequest struct {
	HTML string `json:"html" binding:"required"`
}

func main() {
	router := gin.Default()

	InitJWTSecret() // Initialize JWT secret from new file

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "check"})
	})

	auth := router.Group("/api")
	auth.Use(JWTAuthMiddleware()) // Use imported middleware
	{
		auth.POST("/to-png", convertHTMLToPNG)
		auth.POST("/to-pdf", convertHTMLToPDF)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default port if not set
	}
	log.Printf("Server listening on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
