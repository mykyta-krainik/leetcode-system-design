package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func createCompetition(c *gin.Context) {
	var competition struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ProblemIDs  []int  `json:"problem_ids"`
		ID          int    `json:"id"`
	}
	if err := c.ShouldBindJSON(&competition); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO competitions (name, description, problem_ids, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`
	err = tx.QueryRow(ctx, query, competition.Name, competition.Description, pq.Array(competition.ProblemIDs)).Scan(&competition.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create competition"})
		return
	}

	eventPayload := map[string]interface{}{
		"id":          competition.ID,
		"name":        competition.Name,
		"description": competition.Description,
		"problem_ids": competition.ProblemIDs,
	}
	payload, _ := json.Marshal(eventPayload)
	eventID := uuid.New().String()

	_, err = tx.Exec(ctx, `INSERT INTO outbox (event_id, event_type, payload) VALUES ($1, $2, $3)`, eventID, "competition_created", payload)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to write to outbox"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(500, gin.H{"error": "Failed to commit transaction"})
		return
	}

	go monitorTimeout(eventID, competition.ID)

	c.JSON(201, gin.H{"competition_id": competition.ID})
}

func getCompetitionProblems(c *gin.Context) {
	id := c.Param("id")
	var problemIDs []int

	query := `SELECT problem_ids FROM competitions WHERE id = $1`
	err := dbPool.QueryRow(ctx, query, id).Scan(&problemIDs)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Competition not found"})
		return
	}

	var problems []map[string]interface{}
	serviceName := "problem_management"

	for _, problemID := range problemIDs {
		problem, err := fetchProblem(problemID, serviceName)

		if err != nil {
			if err.Error() == "rate limit exceeded, request queued" {
				problem = map[string]interface{}{
					"error": fmt.Sprintf("Problem ID %d request queued due to rate limiting", problemID),
				}

				log.Printf("Problem ID %d request queued due to rate limiting", problemID)
				continue
			}

			log.Printf("Error fetching problem ID %d: %v", problemID, err)

			continue
		}
		problems = append(problems, problem)
	}

	c.JSON(http.StatusOK, problems)
}

func getCompetition(c *gin.Context) {
	id := c.Param("id")

	cacheKey := fmt.Sprintf("competition:%s", id)
	cachedCompetition, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil && cachedCompetition != "" {
		var competition Competition
		if err := json.Unmarshal([]byte(cachedCompetition), &competition); err == nil {
			c.JSON(http.StatusOK, competition)
			return
		}
	}

	var competition Competition
	query := `SELECT id, name, description, problem_ids, created_at, updated_at FROM competitions WHERE id = $1`
	err = dbPool.QueryRow(ctx, query, id).Scan(
		&competition.ID, &competition.Name, &competition.Description,
		pq.Array(&competition.ProblemIDs), &competition.CreatedAt, &competition.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Competition not found"})
		return
	}

	competitionJSON, err := json.Marshal(competition)
	if err == nil {
		rdb.Set(ctx, cacheKey, competitionJSON, 10*time.Minute)
	}

	c.JSON(http.StatusOK, competition)
}

func getCompetitions(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, name, description, problem_ids, created_at, updated_at FROM competitions")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch competitions"})
		return
	}
	defer rows.Close()

	var competitions []Competition
	for rows.Next() {
		var competition Competition
		if err := rows.Scan(&competition.ID, &competition.Name, &competition.Description, &competition.ProblemIDs, &competition.CreatedAt, &competition.UpdatedAt); err == nil {
			competitions = append(competitions, competition)
		}
	}

	c.JSON(http.StatusOK, competitions)
}
