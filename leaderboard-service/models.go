package main

import "time"

type Leaderboard struct {
	ID            int       `json:"id"`
	CompetitionID int       `json:"competition_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
