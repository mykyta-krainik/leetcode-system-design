package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
	"log"
	"os"
	"sync"
)

var (
	dbPool          *pgxpool.Pool
	rdb             *redis.Client
	ctx             = context.Background()
	rabbitMQConn    *amqp.Connection
	rabbitMQChannel *amqp.Channel
	timeoutRegistry = make(map[string]chan bool)
	mutex           = &sync.Mutex{}
)

func initRabbitMQ() {
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	var err error
	rabbitMQConn, err = amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v\n", err)
	}

	rabbitMQChannel, err = rabbitMQConn.Channel()
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ channel: %v\n", err)
	}

	_, err = rabbitMQChannel.QueueDeclare(
		"competition_created", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare RabbitMQ queue: %v\n", err)
	}

	err = rabbitMQChannel.ExchangeDeclare(
		"rollback_exchange",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare rollback exchange: %v\n", err)
	}

	createAndBindQueue("rollback_events", "rollback_exchange")

	_, err = rabbitMQChannel.QueueDeclare(
		"leaderboard_success", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare leaderboard_success queue: %v\n", err)
	}
}

func createAndBindQueue(queueName string, exchangeName string) {
	_, err := rabbitMQChannel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue %s: %v\n", queueName, err)
	}

	err = rabbitMQChannel.QueueBind(
		queueName,
		"",
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue %s to exchange %s: %v\n", queueName, exchangeName, err)
	}
}

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
		redisAddr = "competition_redis:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v\n", err)
	}
}

func startMessageConsumers() {
	go consumeMessages("leaderboard_success")
	go consumeMessages("rollback_events")
}

func main() {
	initDB()
	initRedis()
	initRabbitMQ()
	initCircuitBreaker()
	defer dbPool.Close()
	defer rdb.Close()
	defer rabbitMQConn.Close()
	defer rabbitMQChannel.Close()

	startMessageConsumers()

	go processOutbox()
	go processInboxMessages()

	r := gin.Default()

	println("Running on port 8080")

	go processQueuedRequests("problem_management", sendRequest)

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/competitions", createCompetition)
	r.GET("/competitions/:id", getCompetition)
	r.GET("/competitions/:id/problems", getCompetitionProblems)
	r.GET("/competitions", getCompetitions)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v\n", err)
	}
}
