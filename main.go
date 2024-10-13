package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte("Hello World"))
}

func main() {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/test", handler)

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	err := server.ListenAndServe()

	if err != nil {
		panic(err)
	}
}
