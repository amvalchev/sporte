package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /event/view/{id}", app.sportEventView)
	mux.HandleFunc("GET /event/create", app.sportEventCreate)
	mux.HandleFunc("POST /event/create", app.sportEventCreatePost)

	return mux
}
