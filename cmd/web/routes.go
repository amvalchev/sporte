package main

import "net/http"

func routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", home)
	mux.HandleFunc("GET /event/view/{id}", sportEventView)
	mux.HandleFunc("GET /event/create", sportEventCreate)
	mux.HandleFunc("POST /event/create", sportEventCreatePost)

	return mux
}
