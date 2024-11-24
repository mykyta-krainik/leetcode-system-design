package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

var cb *gobreaker.CircuitBreaker
var breakerState string

func initCircuitBreaker() {
	settings := gobreaker.Settings{
		Name:        "ProblemManagementCircuitBreaker",
		Timeout:     5 * time.Second,
		MaxRequests: 5,
		Interval:    60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			switch to {
			case gobreaker.StateOpen:
				breakerState = "open"
			case gobreaker.StateHalfOpen:
				breakerState = "half-open"
			case gobreaker.StateClosed:
				breakerState = "closed"
			}
		},
	}
	cb = gobreaker.NewCircuitBreaker(settings)
}

func fetchProblem(problemID int) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("problem:%d", problemID)

	if breakerState == "open" {
		val, err := rdb.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			var cachedProblem map[string]interface{}
			json.Unmarshal([]byte(val), &cachedProblem)
			return cachedProblem, nil
		}
	}

	problemData, err := cb.Execute(func() (interface{}, error) {
		url := fmt.Sprintf("http://problem_management:8080/problems/%d", problemID)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch problem: %s", resp.Status)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var problem map[string]interface{}
		if err := json.Unmarshal(body, &problem); err != nil {
			return nil, err
		}

		rdb.Set(ctx, cacheKey, body, 10*time.Minute)

		return problem, nil
	})

	if err != nil {
		if breakerState == "closed" {
			return nil, err
		}

		return nil, fmt.Errorf("service unavailable and no cached data found")
	}

	return problemData.(map[string]interface{}), nil
}
