package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

const (
	clientRateLimit   = 4
	clientWindow      = time.Minute
	clientBucket      = 10 * time.Second
	clientBucketCount = int(clientWindow / clientBucket)
)

func rateLimiter(serviceName string, checkOnly bool) (bool, error) {
	now := time.Now().Unix()
	currentBucket := now / int64(clientBucket.Seconds())
	redisKey := fmt.Sprintf("outgoing_limit:%s:%d", serviceName, currentBucket)

	totalRequests := int64(0)
	for i := 0; i < clientBucketCount; i++ {
		bucketKey := fmt.Sprintf("outgoing_limit:%s:%d", serviceName, currentBucket-int64(i))
		bucketCount, _ := rdb.Get(ctx, bucketKey).Int64()
		totalRequests += bucketCount

		log.Printf("Bucket %s: %d", bucketKey, bucketCount)
	}

	log.Printf("Total requests: %d", totalRequests)

	if totalRequests >= clientRateLimit {
		return false, nil
	}

	if !checkOnly {
		_, err := rdb.Incr(ctx, redisKey).Result()
		if err != nil {
			return false, err
		}

		rdb.Expire(ctx, redisKey, clientWindow)
	}

	return true, nil
}

func enqueueRequest(serviceName string, request string) error {
	queueKey := fmt.Sprintf("request_queue:%s", serviceName)
	err := rdb.LPush(ctx, queueKey, request).Err()

	return err
}

func dequeueRequest(serviceName string) (string, error) {
	queueKey := fmt.Sprintf("request_queue:%s", serviceName)
	request, err := rdb.RPop(ctx, queueKey).Result()

	return request, err
}

func processQueuedRequests(serviceName string, sendRequest func(request string, serviceName string) error) {
	for {
		canSend, err := rateLimiter(serviceName, true)
		if err != nil || !canSend {
			time.Sleep(1 * time.Second)

			continue
		}

		request, err := dequeueRequest(serviceName)
		if err == redis.Nil {
			time.Sleep(1 * time.Second)

			continue
		}

		if err != nil {
			log.Printf("Error dequeuing request: %v", err)
			continue
		}

		if err := sendRequest(request, serviceName); err != nil {
			log.Printf("Error sending request: %v", err)
			enqueueRequest(serviceName, request)
		}
	}
}

func sendRequest(request string, serviceName string) error {
	var problemID int
	_, err := fmt.Sscanf(request, "problem_id:%d", &problemID)

	if err != nil {
		return fmt.Errorf("invalid request format: %v", err)
	}

	log.Printf("Sending request for problem ID %d", problemID)

	problem, err := fetchProblem(problemID, serviceName)

	log.Printf("Fetched problem ID %d: %v", problemID, problem)

	return err
}
