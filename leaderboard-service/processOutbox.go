package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
	"time"
)

func processOutbox() {
	for {
		rows, err := dbPool.Query(ctx, "SELECT id, event_id, event_type, payload, retries FROM outbox WHERE processed = FALSE AND retries < $1 LIMIT 10", maxRetries)
		if err != nil {
			log.Printf("Failed to fetch outbox events: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for rows.Next() {
			var id, retries int
			var eventID, eventType string
			var payload []byte

			if err := rows.Scan(&id, &eventID, &eventType, &payload, &retries); err != nil {
				log.Printf("Failed to scan outbox row: %v\n", err)
				continue
			}

			var payloadMap map[string]interface{}

			if err := json.Unmarshal(payload, &payloadMap); err != nil {
				log.Printf("Failed to unmarshal payload for event ID %s: %v\n", eventID, err)
				continue
			}

			event := map[string]interface{}{
				"event_id":   eventID,
				"event_type": eventType,
				"payload":    payloadMap,
			}

			eventPayload, err := json.Marshal(event)

			if err != nil {
				log.Printf("Failed to marshal outbox event: %v\n", err)
				continue
			}

			err = rabbitMQChannel.Publish(
				"", "leaderboard_success", false, false,
				amqp.Publishing{ContentType: "application/json", Body: eventPayload},
			)
			if err != nil {
				log.Printf("Failed to publish outbox event (attempt %d/%d): %v\n", retries+1, maxRetries, err)
				_, updateErr := dbPool.Exec(ctx, "UPDATE outbox SET retries = retries + 1 WHERE id = $1", id)
				if updateErr != nil {
					log.Printf("Failed to update retries for outbox event ID %s: %v\n", eventID, updateErr)
				}
				continue
			}

			_, err = dbPool.Exec(ctx, "UPDATE outbox SET processed = TRUE WHERE id = $1", id)
			if err != nil {
				log.Printf("Failed to mark outbox event as processed: %v\n", err)
			}
		}

		rows.Close()
		time.Sleep(1 * time.Second)
	}
}
