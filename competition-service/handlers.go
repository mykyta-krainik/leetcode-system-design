// handlers.go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func createCompetition(c *gin.Context) {
	var competition Competition
	if err := c.ShouldBindJSON(&competition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `INSERT INTO competitions (name, description, problem_ids, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`
	err := dbPool.QueryRow(
		ctx,
		query,
		competition.Name,
		competition.Description,
		pq.Array(competition.ProblemIDs),
	).Scan(&competition.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create competition"})
		return
	}

	c.JSON(http.StatusCreated, competition)
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
	for _, problemID := range problemIDs {
		problem, err := fetchProblem(problemID)
		if err == nil {
			problems = append(problems, problem)
		}
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
