package main

import (
	"encoding/json"
	"log"
	"time"
)

func consumeMessages(queueName string) {
	msgs, err := rabbitMQChannel.Consume(
		queueName, // Queue name
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register consumer for queue %s: %v\n", queueName, err)
	}

	for msg := range msgs {
		log.Printf("Message received: %s\n", msg.Body)

		var message map[string]interface{}
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("Failed to parse message from queue %s: %v\n", queueName, err)
			continue
		}

		eventID := message["event_id"].(string)
		payload := message["payload"]

		payloadStr, err := json.Marshal(payload)

		if err != nil {
			log.Printf("Failed to marshal payload for event_id %s in queue %s: %v\n", eventID, queueName, err)
			continue
		}

		_, err = dbPool.Exec(
			ctx,
			`INSERT INTO inbox (event_id, event_type, payload) VALUES ($1, $2, $3) ON CONFLICT (event_id) DO NOTHING`,
			eventID, queueName, payloadStr,
		)

		if err != nil {
			log.Printf("Failed to insert message into inbox for queue %s: %v\n", queueName, err)
		}
	}
}

func processInboxMessages() {
	for {
		rows, err := dbPool.Query(ctx, "SELECT id, event_id, event_type, payload, retries FROM inbox WHERE processed = FALSE AND retries < $1 LIMIT 10", maxRetries)
		if err != nil {
			log.Printf("Failed to fetch inbox messages: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for rows.Next() {
			var id, retries int
			var eventID, eventType string
			var payload []byte

			if err := rows.Scan(&id, &eventID, &eventType, &payload, &retries); err != nil {
				log.Printf("Failed to scan inbox row: %v\n", err)
				continue
			}

			var processErr error
			switch eventType {
			case "competition_created":
				processErr = handleCompetitionCreated(payload)
			case "leaderboard_rollback_queue":
				processErr = handleRollback(payload)
			default:
				log.Printf("Unknown event type: %s\n", eventType)
				continue
			}

			if processErr != nil {
				log.Printf("Failed to process inbox event (attempt %d/%d): %v\n", retries+1, maxRetries, processErr)
				_, updateErr := dbPool.Exec(ctx, "UPDATE inbox SET retries = retries + 1 WHERE id = $1", id)
				if updateErr != nil {
					log.Printf("Failed to update retries for inbox event ID %s: %v\n", eventID, updateErr)
				}
				continue
			}

			_, err = dbPool.Exec(ctx, "UPDATE inbox SET processed = TRUE, processed_at = NOW() WHERE id = $1", id)
			if err != nil {
				log.Printf("Failed to mark inbox event as processed: %v\n", err)
			}
		}

		rows.Close()
		time.Sleep(1 * time.Second)
	}
}
