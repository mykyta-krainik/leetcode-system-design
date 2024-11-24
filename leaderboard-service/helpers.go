package main

import (
	"encoding/json"
	"github.com/google/uuid"
)

func handleCompetitionCreated(payload []byte) error {
	var event struct {
		CompetitionID int `json:"id"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	_, err := dbPool.Exec(ctx, "INSERT INTO leaderboards (competition_id, created_at, updated_at) VALUES ($1, NOW(), NOW())", event.CompetitionID)
	if err != nil {
		return err
	}

	eventId := uuid.New().String()

	successEvent := map[string]interface{}{
		"event_id":       eventId,
		"competition_id": event.CompetitionID,
	}

	payload, _ = json.Marshal(successEvent)
	_, err = dbPool.Exec(ctx, "INSERT INTO outbox (event_id, event_type, payload) VALUES ($1, $2, $3)", uuid.New().String(), "leaderboard_success", payload)
	return err
}

func handleRollback(payload []byte) error {
	var event struct {
		CompetitionID int `json:"competition_id"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	_, err := dbPool.Exec(ctx, "DELETE FROM leaderboards WHERE competition_id = $1", event.CompetitionID)
	return err
}
