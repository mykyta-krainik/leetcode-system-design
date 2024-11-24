package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func getLeaderboards(c *gin.Context) {
	rows, err := dbPool.Query(ctx, "SELECT id, competition_id, created_at, updated_at FROM leaderboards")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboards"})
		return
	}
	defer rows.Close()

	var leaderboards []Leaderboard
	for rows.Next() {
		var leaderboard Leaderboard
		if err := rows.Scan(&leaderboard.ID, &leaderboard.CompetitionID, &leaderboard.CreatedAt, &leaderboard.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan leaderboards"})
			return
		}
		leaderboards = append(leaderboards, leaderboard)
	}

	c.JSON(http.StatusOK, leaderboards)
}

func getLeaderboard(c *gin.Context) {
	id := c.Param("id")
	var leaderboard Leaderboard

	err := dbPool.QueryRow(ctx,
		"SELECT id, competition_id, created_at, updated_at FROM leaderboards WHERE id = $1",
		id,
	).Scan(&leaderboard.ID, &leaderboard.CompetitionID, &leaderboard.CreatedAt, &leaderboard.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leaderboard not found"})
		return
	}

	c.JSON(http.StatusOK, leaderboard)
}
