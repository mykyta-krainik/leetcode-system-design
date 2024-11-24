package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/streadway/amqp"
)

var (
	dbPool          *pgxpool.Pool
	ctx             = context.Background()
	rabbitMQConn    *amqp.Connection
	rabbitMQChannel *amqp.Channel
)

const maxRetries = 5

func initDB() {
	dbURL := os.Getenv("DATABASE_URL")
	var err error
	dbPool, err = pgxpool.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
}

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
		log.Fatalf("Failed to declare competition_events queue: %v\n", err)
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

	createAndBindQueue("leaderboard_rollback_queue", "rollback_exchange")

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

func startMessageConsumers() {
	go consumeMessages("competition_created")
	go consumeMessages("leaderboard_rollback_queue")
}

func main() {
	initDB()
	initRabbitMQ()

	defer dbPool.Close()
	defer rabbitMQConn.Close()
	defer rabbitMQChannel.Close()

	startMessageConsumers()

	go processInboxMessages()
	go processOutbox()

	r := gin.Default()
	r.GET("/leaderboards/:id", getLeaderboard)
	r.GET("/leaderboards", getLeaderboards)

	log.Println("Leaderboard Service running on port 8081")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v\n", err)
	}
}
