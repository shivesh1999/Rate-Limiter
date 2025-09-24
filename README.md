# Token Bucket Rate Limiter

A high-performance, Redis-backed rate limiting service implemented in Go using the token bucket algorithm. This service provides a flexible and scalable solution for API rate limiting based on client IP addresses.

## Features

- **Token Bucket Algorithm**: Efficient rate limiting with token refill over time
- **Redis Backend**: Distributed rate limiting with persistence
- **IP-Based Limiting**: Automatically identifies and limits by client IP
- **Configurable Parameters**: Customize token capacity, refill rate, and TTL
- **Docker Support**: Easy deployment with Docker and Docker Compose
- **Gin Middleware**: Simple integration with Gin web applications

## Architecture

The service implements the token bucket algorithm with the following components:

- **Token Bucket**: Each client IP gets a bucket with a maximum token capacity
- **Token Consumption**: Each request consumes one token
- **Token Refill**: Tokens are refilled at a configurable rate over time
- **Redis Storage**: Buckets are stored in Redis for distributed deployments
- **TTL Support**: Bucket data expires after a configurable time period

## Installation

### Using Docker Compose (Recommended)

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/token-bucket-rate-limiter.git
   cd token-bucket-rate-limiter
   ```

2. Configure the environment variables in `.env` file (or use the defaults)

3. Start the services:
   ```bash
   docker-compose up -d
   ```

The rate limiter will be available at `http://localhost:8080`.

### Manual Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/token-bucket-rate-limiter.git
   cd token-bucket-rate-limiter
   ```

2. Ensure you have Go 1.24 or later installed

3. Install dependencies:
   ```bash
   go mod tidy
   ```

4. Configure the environment variables in `.env` file

5. Start a Redis server:
   ```bash
   # Install Redis if needed
   # Then start Redis server
   redis-server
   ```

6. Build and run the application:
   ```bash
   go build -o rate-limiter .
   ./rate-limiter
   ```

## Configuration

The application is configured using environment variables, which can be set in the `.env` file:

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_HOST` | Redis server hostname/IP | localhost |
| `REDIS_PORT` | Redis server port | 6379 |
| `RATE_LIMIT` | Maximum tokens per IP | 10 |
| `REFILL_RATE` | Tokens added per second | 1 |
| `TTL_SECONDS` | Time-to-live for bucket entries (seconds) | 3600 |
| `SUCCESS_URL` | Redirect URL for allowed requests | https://google.com |

## Usage

### As a Standalone Service

The rate limiter runs as a standalone service that can protect any backend API or website. When a request is received:

1. If the request is allowed (within rate limits), it redirects to the `SUCCESS_URL`
2. If the request exceeds the rate limit, it returns a 429 (Too Many Requests) status with an error message

### As a Middleware in Your Go Application

You can integrate the rate limiter into your own Go application:

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"yourusername/token-bucket-rate-limiter/limiter"
)

func main() {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	// Initialize rate limiter
	rateLimiter := limiter.NewRedisTokenBucket(
		client,
		10,    // capacity
		1,     // refill rate
		3600,  // TTL in seconds
	)

	// Create Gin router
	r := gin.Default()

	// Apply rate limiting middleware
	r.Use(func(c *gin.Context) {
		ip := c.ClientIP()
		if !rateLimiter.AllowRequest(c.Request.Context(), ip) {
			c.JSON(429, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})

	// Define your routes
	r.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Success"})
	})

	r.Run(":8080")
}
```

## API Endpoints

The service exposes the following endpoint:

- `GET /api`: A sample endpoint protected by rate limiting
  - Returns 200 OK with a success message if the request is allowed
  - Returns 429 Too Many Requests if the rate limit is exceeded

## Testing

The project includes unit tests for the rate limiter functionality. To run the tests:

```bash
go test -v ./limiter
```

The tests verify:
- Basic rate limiting functionality
- Token refill behavior
- TTL expiration

## License

[MIT License](LICENSE)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.