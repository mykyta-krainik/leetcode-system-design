package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"os"
)

var (
	dbPool *pgxpool.Pool
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

func main() {
	initDB()

	defer dbPool.Close()

	r := gin.Default()
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/problems", createProblem)
	r.GET("/problems/:id", getProblem)
	r.GET("/problems", getAllProblems)
	r.GET("/problems/filter", filterProblems)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v\n", err)
	}
}
