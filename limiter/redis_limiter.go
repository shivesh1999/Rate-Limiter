package limiter

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisTokenBucket defines the rate limiter structure
type RedisTokenBucket struct {
	client     *redis.Client
	capacity   float64       // Max tokens per IP
	refillRate float64       // Tokens added per second
	ttl        time.Duration // Expiration time for IP entries
}

// NewRedisTokenBucket initializes a Redis-backed token bucket with TTL support
func NewRedisTokenBucket(client *redis.Client, capacity, refillRate float64, ttl time.Duration) *RedisTokenBucket {
	return &RedisTokenBucket{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		ttl:        ttl,
	}
}

// getIPKey generates a Redis key for an IP
func (rtb *RedisTokenBucket) getIPKey(ip string) string {
	return fmt.Sprintf("rate_limit:ip:%s", ip)
}

// AllowRequest checks if an IP can proceed with a request
func (rtb *RedisTokenBucket) AllowRequest(ctx context.Context, ip string) bool {
	redisKey := rtb.getIPKey(ip)
	now := time.Now().Unix()

	// Fetch IP-specific rate limit data from Redis
	tokens, err := rtb.client.Get(ctx, redisKey+":tokens").Float64()
	if err != nil {
		tokens = rtb.capacity // Default to full bucket if key doesn't exist
	}

	lastUpdated, err := rtb.client.Get(ctx, redisKey+":last_updated").Int64()
	if err != nil {
		lastUpdated = now
	}

	// Calculate elapsed time and refill tokens
	elapsed := float64(now - lastUpdated)
	newTokens := math.Min(rtb.capacity, tokens+(elapsed*rtb.refillRate))

	// Allow request only if at least 1 token is available
	if newTokens >= 1 {
		pipe := rtb.client.TxPipeline()
		pipe.Set(ctx, redisKey+":tokens", newTokens-1, rtb.ttl) // Set TTL for cleanup
		pipe.Set(ctx, redisKey+":last_updated", now, rtb.ttl)   // Ensure timestamp also expires
		_, err = pipe.Exec(ctx)
		if err != nil {
			log.Printf("[RedisLimiter] Error updating Redis: %v", err)
			return false
		}
		return true
	}

	// Log rejections
	log.Printf("[RedisLimiter] Request rejected for IP %s. Tokens left: %.2f", ip, newTokens)
	return false
}
