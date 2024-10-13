package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/mykyta-krainik/leetcode-system-design/models"
)

var (
	problems = make(map[string]*models.Problem)
	mu       sync.Mutex
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/problems", func(r chi.Router) {
		r.Post("/", createProblem)
		r.Get("/", getAllProblems)
		r.Get("/{problemID}", getProblem)
		r.Put("/{problemID}", updateProblem)
		r.Delete("/{problemID}", deleteProblem)
	})

	log.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", r)
}

func createProblem(w http.ResponseWriter, r *http.Request) {
	var p models.Problem
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	mu.Lock()
	problems[p.ID] = &p
	mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func getAllProblems(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	result := make([]*models.Problem, 0, len(problems))
	for _, p := range problems {
		result = append(result, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getProblem(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemID")

	mu.Lock()
	problem, exists := problems[problemID]
	mu.Unlock()

	if !exists {
		http.Error(w, "Problem not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(problem)
}

func updateProblem(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemID")

	var updatedProblem models.Problem
	if err := json.NewDecoder(r.Body).Decode(&updatedProblem); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	mu.Lock()
	problem, exists := problems[problemID]
	if !exists {
		mu.Unlock()
		http.Error(w, "Problem not found", http.StatusNotFound)
		return
	}

	problem.Title = updatedProblem.Title
	problem.Description = updatedProblem.Description
	problem.Difficulty = updatedProblem.Difficulty
	problem.Tags = updatedProblem.Tags
	problem.UpdatedAt = time.Now()

	problems[problemID] = problem
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(problem)
}

func deleteProblem(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemID")

	mu.Lock()
	defer mu.Unlock()

	if _, exists := problems[problemID]; !exists {
		http.Error(w, "Problem not found", http.StatusNotFound)
		return
	}

	delete(problems, problemID)
	w.WriteHeader(http.StatusNoContent)
}
