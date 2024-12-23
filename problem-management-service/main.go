package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"os"
)

var (
	dbPool *pgxpool.Pool
	rdb    *redis.Client
	ctx    = context.Background()
)

func initDB() {
	dbURL := os.Getenv("DATABASE_URL")
	var err error
	dbPool, err = pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
}

func initRedis() {
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "problem_management_redis:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v\n", err)
	}
}

func main() {
	initDB()
	initRedis()

	defer dbPool.Close()
	defer rdb.Close()

	r := gin.Default()

	r.Use(rateLimiterMiddleware())

	//r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/problems", createProblem)
	r.GET("/problems/:id", getProblem)
	r.GET("/problems", getAllProblems)
	r.GET("/problems/filter", filterProblems)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v\n", err)
	}
}
