package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Server", "Go")
	w.Write([]byte("Hello from Sporte"))
}

func sportEventView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "Displat a specific sport event with ID %d...", id)
}

func sportEventCreate(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Display a form for createing a new sport event..."))
}

func sportEventCreatePost(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Save a new sport event..."))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", home)
	mux.HandleFunc("GET /event/view/{id}", sportEventView)
	mux.HandleFunc("GET /event/create", sportEventCreate)
	mux.HandleFunc("POST /event/create", sportEventCreatePost)

	log.Print("starting server on :8000")

	err := http.ListenAndServe(":8000", mux)
	log.Fatal(err)
}
