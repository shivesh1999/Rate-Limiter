package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestAllowRequest_BasicFunctionality(t *testing.T) {
	ctx := context.Background()

	// Initialize Redis client (use a different DB for testing)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use a separate DB for testing
	})
	defer client.FlushDB(ctx) // Cleanup after test

	// Initialize rate limiter with small limits for testing
	rateLimiter := NewRedisTokenBucket(client, 3, 1, 10*time.Second)

	ip := "192.168.1.100"

	// First 3 requests should pass
	assert.True(t, rateLimiter.AllowRequest(ctx, ip), "Expected request 1 to pass")
	assert.True(t, rateLimiter.AllowRequest(ctx, ip), "Expected request 2 to pass")
	assert.True(t, rateLimiter.AllowRequest(ctx, ip), "Expected request 3 to pass")

	// 4th request should be rejected
	assert.False(t, rateLimiter.AllowRequest(ctx, ip), "Expected request 4 to be rate limited")

	// Wait for token refill (1 second)
	time.Sleep(1 * time.Second)

	// Now 1 request should pass again
	assert.True(t, rateLimiter.AllowRequest(ctx, ip), "Expected request after refill to pass")
}

func TestAllowRequest_TTLExpiry(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.FlushDB(ctx)

	rateLimiter := NewRedisTokenBucket(client, 2, 1, 5*time.Second)
	ip := "192.168.1.200"

	// Consume all tokens
	assert.True(t, rateLimiter.AllowRequest(ctx, ip))
	assert.True(t, rateLimiter.AllowRequest(ctx, ip))
	assert.False(t, rateLimiter.AllowRequest(ctx, ip)) // Should be blocked

	// Wait for TTL to expire
	time.Sleep(6 * time.Second)

	// Now IP should be reset, and requests should pass again
	assert.True(t, rateLimiter.AllowRequest(ctx, ip), "Expected request after TTL expiry to pass")
}
