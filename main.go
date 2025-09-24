package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"token-bucket-rate-limiter/limiter"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

func getEnvOrFatal(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.WithField("env_var", key).Fatal("Required environment variable is missing")
	}
	return val
}

func init() {
	// Configure logrus
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.WithField("error", err).Fatal("Error loading .env file")
	}
}

// Middleware function to handle rate limiting
func RateLimitMiddleware(rateLimiter *limiter.RedisTokenBucket, successURL, failureMessage string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ip := c.ClientIP()

		if ip == "" {
			log.WithField("path", c.Request.URL.Path).Error("Unable to determine IP")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to determine IP"})
			c.Abort()
			return
		}

		allowed := rateLimiter.AllowRequest(ctx, ip)
		logger := log.WithFields(log.Fields{
			"ip":   ip,
			"path": c.Request.URL.Path,
		})

		if !allowed {
			logger.Info("Rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": failureMessage,
				"ip":    ip,
			})
			c.Abort()
			return
		}

		logger.Info("Request allowed")
		c.Redirect(http.StatusTemporaryRedirect, successURL)
		c.Abort()
	}
}

func main() {
	// Get and validate environment variables
	redisHost := getEnvOrFatal("REDIS_HOST")
	redisPort := getEnvOrFatal("REDIS_PORT")
	rateLimitStr := getEnvOrFatal("RATE_LIMIT")
	refillRateStr := getEnvOrFatal("REFILL_RATE")
	ttlSecondsStr := getEnvOrFatal("TTL_SECONDS")
	successURL := getEnvOrFatal("SUCCESS_URL")
	failureMessage := "Rate limit exceeded. Please try again later."

	// Parse numeric values
	rateLimit, err := strconv.ParseFloat(rateLimitStr, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"value": rateLimitStr,
		}).Fatal("Invalid RATE_LIMIT")
	}

	refillRate, err := strconv.ParseFloat(refillRateStr, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"value": refillRateStr,
		}).Fatal("Invalid REFILL_RATE")
	}

	ttlSeconds, err := strconv.Atoi(ttlSecondsStr)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"value": ttlSecondsStr,
		}).Fatal("Invalid TTL_SECONDS")
	}

	// Initialize Redis client with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB:   0,
	})

	// Test Redis connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.WithField("error", err).Fatal("Failed to connect to Redis")
	}

	// Initialize rate limiter
	rateLimiter := limiter.NewRedisTokenBucket(client, rateLimit, refillRate, time.Duration(ttlSeconds)*time.Second)

	// Create Gin router in release mode
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Apply rate limiting middleware with redirect and custom failure message
	r.Use(RateLimitMiddleware(rateLimiter, successURL, failureMessage))

	// Define API routes
	r.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Request processed successfully!",
		})
	})

	// Add health check endpoint
	r.GET("/health", func(c *gin.Context) {
		if err := client.Ping(c.Request.Context()).Err(); err != nil {
			log.WithField("error", err).Error("Health check failed")
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "reason": "Redis connection failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Configure server with timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.WithField("error", err).Fatal("Server forced to shutdown")
		}
	}()

	log.Info("Server is running on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithField("error", err).Fatal("Server startup failed")
	}
}
