package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	rateLimit       = 5
	windowDuration  = time.Minute
	bucketInterval  = 10 * time.Second
	numberOfBuckets = int(windowDuration / bucketInterval)
)

func rateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetHeader("X-Client-ID")

		log.Printf("Client ID: %s", clientID)

		if clientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Client-ID header"})
			c.Abort()
			return
		}

		now := time.Now().Unix()
		currentBucket := now / int64(bucketInterval.Seconds())
		log.Printf("Current bucket: %d", currentBucket)

		totalRequests := int64(0)

		for i := 0; i < numberOfBuckets; i++ {
			bucketKey := fmt.Sprintf("rate_limit:%s:%d", clientID, currentBucket-int64(i))
			bucketCount, _ := rdb.Get(ctx, bucketKey).Int64()
			log.Printf("Bucket %d: %d", currentBucket-int64(i), bucketCount)
			totalRequests += bucketCount
		}

		log.Printf("Total requests: %d", totalRequests)

		if totalRequests >= rateLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":  "Rate limit exceeded",
				"limit":  rateLimit,
				"window": windowDuration.Seconds(),
			})
			c.Abort()
			return
		}

		redisKey := fmt.Sprintf("rate_limit:%s:%d", clientID, currentBucket)
		_, err := rdb.Incr(ctx, redisKey).Result()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			c.Abort()
			return
		}

		rdb.Expire(ctx, redisKey, windowDuration)

		c.Next()
	}
}
