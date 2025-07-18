package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var JWT_SECRET_KEY []byte

func InitJWTSecret() {
	if os.Getenv("JWT_SECRET_KEY") == "" {
		if err := godotenv.Load(); err != nil {
			log.Printf("[ERROR] Error loading .env file: %v\n", err)
		}
	}
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Printf("[ERROR] JWT_SECRET_KEY environment variable is not set")
	}
	JWT_SECRET_KEY = []byte(secret)
}

// JWTAuthMiddleware is a custom Gin middleware to validate JWT tokens.
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && strings.ToLower(parts[0]) == "bearer") {
			c.JSON(401, gin.H{"error": "Invalid Authorization header format. Expected 'Bearer <token>'"})
			c.Abort()
			return
		}
		tokenString := parts[1]

		_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return JWT_SECRET_KEY, nil
		})
		if err != nil {
			switch err {
			case jwt.ErrTokenMalformed:
				c.JSON(401, gin.H{"error": "That's not even a token"})
			case jwt.ErrTokenExpired:
				c.JSON(401, gin.H{"error": "Token is expired"})
			case jwt.ErrTokenNotValidYet:
				c.JSON(401, gin.H{"error": "Token not active yet"})
			default:
				c.JSON(401, gin.H{"error": fmt.Sprintf("Invalid token: %v", err)})
			}
			c.Abort()
			return
		}
		c.Next()
	}
}

// GenerateJWT generates a JWT token for a given user ID and expiration duration.
func GenerateJWT(userID string, duration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"id":  userID,
		"exp": time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWT_SECRET_KEY)
}
