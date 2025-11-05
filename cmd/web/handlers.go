package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/amvalchev/sporte/internal/models"
)

type templateData struct {
	HomeViewEvents []models.EventHomeView
	Event          models.SportEvent
	Events         []models.SportEvent
	Sport          models.Sport
	Venue          models.Venue
	Teams          []models.TeamInEvent
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Server", "Go")

	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/home.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	events, err := app.sportEvents.Latest()
	if err != nil {
		log.Print(err.Error())
		return
	}

	data := templateData{
		HomeViewEvents: events,
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *application) sportEventView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	// Call the new Get() method, which returns three values.
	event, sport, venue, teams, err := app.sportEvents.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/view.tmpl",
	}
	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Populate the new fields in our templateData struct.
	data := templateData{
		Event: event,
		Sport: sport,
		Venue: venue,
		Teams: teams,
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *application) sportEventCreate(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/create.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = ts.ExecuteTemplate(w, "base", nil)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (app *application) sportEventCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	eventName := r.PostForm.Get("eventName")
	description := r.PostForm.Get("description")
	eventDateTimeStr := r.PostForm.Get("eventDateTime")
	sportIDStr := r.PostForm.Get("sportID")
	venueIDStr := r.PostForm.Get("venueID")

	sportID, err := strconv.Atoi(sportIDStr)
	if err != nil {
		http.Error(w, "Invalid Sport ID", http.StatusBadRequest)
		return
	}
	venueID, err := strconv.Atoi(venueIDStr)
	if err != nil {
		http.Error(w, "Invalid Venue ID", http.StatusBadRequest)
		return
	}

	// --- THIS IS THE FINAL FIX ---
	// This layout perfectly matches the browser's "datetime-local" input: "2025-11-15T16:36"
	layout := "2006-01-02T15:04"
	eventDateTime, err := time.Parse(layout, eventDateTimeStr)
	if err != nil {
		log.Printf("Failed to parse date string from form '%s'. Error: %v", eventDateTimeStr, err)
		http.Error(w, "Invalid Date format.", http.StatusBadRequest)
		return
	}

	// The rest of the function is correct.
	id, err := app.sportEvents.Insert(eventName, eventDateTime, description, sportID, venueID)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	redirectURL := fmt.Sprintf("/event/view/%d", id)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
