package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"log"
	"strconv"
	"time"
)

func monitorTimeout(eventID string, competitionID int) {
	cancelChan := make(chan bool)
	mutex.Lock()
	timeoutRegistry[strconv.Itoa(competitionID)] = cancelChan
	mutex.Unlock()

	select {
	case <-time.After(10 * time.Second):
		log.Printf("Timeout expired for competition ID %d. Initiating rollback.\n", competitionID)
		initiateRollback(eventID, competitionID)
	case <-cancelChan:
		log.Printf("Timeout canceled for competition ID %d.\n", competitionID)
	}

	mutex.Lock()
	delete(timeoutRegistry, eventID)
	mutex.Unlock()
}

func initiateRollback(eventID string, competitionID int) {
	eventID = uuid.New().String()

	rollbackEvent := map[string]interface{}{
		"event_id":   eventID,
		"event_type": "rollback_events",
		"payload": map[string]interface{}{
			"competition_id": competitionID,
			"reason":         "Timeout expired",
		},
	}

	payload, _ := json.Marshal(rollbackEvent)

	err := rabbitMQChannel.Publish(
		"rollback_exchange", "", false, false,
		amqp.Publishing{ContentType: "application/json", Body: payload},
	)
	if err != nil {
		log.Printf("Failed to publish rollback event for competition ID %d: %v\n", competitionID, err)
	} else {
		log.Printf("Rollback event published for competition ID %d.\n", competitionID)
	}
}

func handleRollback(payload []byte) error {
	var rollbackEvent struct {
		EventID       string `json:"event_id"`
		CompetitionID int    `json:"competition_id"`
		Reason        string `json:"reason"`
	}

	if err := json.Unmarshal(payload, &rollbackEvent); err != nil {
		return err
	}

	_, err := dbPool.Exec(ctx, "DELETE FROM competitions WHERE id = $1", rollbackEvent.CompetitionID)
	if err != nil {
		return err
	}

	log.Printf("Rollback completed for competition ID %d: %s\n", rollbackEvent.CompetitionID, rollbackEvent.Reason)
	return nil
}

func handleLeaderboardSuccess(payload []byte) error {
	var successEvent struct {
		EventID       string `json:"event_id"`
		CompetitionID int    `json:"competition_id"`
	}

	if err := json.Unmarshal(payload, &successEvent); err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	if cancelChan, exists := timeoutRegistry[strconv.Itoa(successEvent.CompetitionID)]; exists {
		cancelChan <- true
		delete(timeoutRegistry, strconv.Itoa(successEvent.CompetitionID))
		log.Printf("Timeout canceled for competition ID %d\n", successEvent.CompetitionID)
	} else {
		log.Printf("No timeout found for competition ID %d\n", successEvent.CompetitionID)
	}
	return nil
}
