package main

import (
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"net/http"
)

func createProblem(c *gin.Context) {
	var problem Problem
	if err := c.ShouldBindJSON(&problem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `INSERT INTO problems (title, description, difficulty, tags, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id`
	err := dbPool.QueryRow(ctx, query, problem.Title, problem.Description, problem.Difficulty, pq.Array(problem.Tags)).Scan(&problem.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	c.JSON(http.StatusCreated, problem)
}

func getProblem(c *gin.Context) {
	id := c.Param("id")

	var problem Problem
	query := `SELECT id, title, description, difficulty, tags, created_at, updated_at FROM problems WHERE id = $1`
	err := dbPool.QueryRow(ctx, query, id).Scan(
		&problem.ID, &problem.Title, &problem.Description, &problem.Difficulty,
		pq.Array(&problem.Tags), &problem.CreatedAt, &problem.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	c.JSON(http.StatusOK, problem)
}

func getAllProblems(c *gin.Context) {
	var problems []Problem

	query := `SELECT id, title, description, difficulty, tags, created_at, updated_at FROM problems`
	rows, err := dbPool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve problems"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var problem Problem
		err := rows.Scan(&problem.ID, &problem.Title, &problem.Description, &problem.Difficulty, pq.Array(&problem.Tags), &problem.CreatedAt, &problem.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse problem data"})
			return
		}
		problems = append(problems, problem)
	}

	c.JSON(http.StatusOK, gin.H{"data": problems})
}

func filterProblems(c *gin.Context) {
	text := c.Query("text")
	tag := c.Query("tag")
	difficulty := c.Query("difficulty")

	var problems []Problem
	query := `SELECT id, title, description, difficulty, created_at, updated_at FROM problems WHERE 
			  (title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%') 
			  AND difficulty = $2 AND $3 = ANY(tags)`
	rows, err := dbPool.Query(ctx, query, text, difficulty, tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve problems"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var problem Problem
		if err := rows.Scan(&problem.ID, &problem.Title, &problem.Description, &problem.Difficulty, &problem.CreatedAt, &problem.UpdatedAt); err == nil {
			problems = append(problems, problem)
		}
	}

	c.JSON(http.StatusOK, problems)
}
