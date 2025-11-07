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

// FIX #1: The templateData struct is now more explicit.
type templateData struct {
	HomeViewEvents []models.EventHomeView
	Event          models.SportEvent
	Sport          models.Sport
	Venue          models.Venue
	TeamsInEvent   []models.TeamInEvent // For the view page (teams with scores/players)

	// Data for the create form dropdowns
	AllSports []models.Sport
	AllVenues []models.Venue
	AllTeams  []models.Team
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/home.tmpl",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	events, err := app.sportEvents.Latest()
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data := templateData{HomeViewEvents: events}
	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) sportEventView(w http.ResponseWriter, r *http.Request) {
	// A bad ID in the URL is a client error, so we return a 404 Not Found.
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	event, sport, venue, teams, err := app.sportEvents.Get(id)
	if err != nil {
		// If the record specifically isn't found, that's also a 404.
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			// Any other database error is a server problem (500).
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data := templateData{
		Event:        event,
		Sport:        sport,
		Venue:        venue,
		TeamsInEvent: teams,
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) sportEventCreate(w http.ResponseWriter, r *http.Request) {
	// Step 1: Fetch all sports
	sports, err := app.sportEvents.GetAllSports()
	if err != nil {
		// If this step fails, log the specific error.
		log.Printf("ERROR fetching sports: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Step 2: Fetch all venues
	venues, err := app.sportEvents.GetAllVenues()
	if err != nil {
		// If this step fails, log the specific error.
		log.Printf("ERROR fetching venues: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Step 3: Fetch all teams
	teams, err := app.sportEvents.GetAllTeams()
	if err != nil {
		// If this step fails, log the specific error.
		log.Printf("ERROR fetching teams: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Step 4: Parse the template files
	files := []string{
		"./ui/html/base.tmpl",
		"./ui/html/partials/nav.tmpl",
		"./ui/html/pages/create.tmpl",
	}
	ts, err := template.ParseFiles(files...)
	if err != nil {
		log.Printf("ERROR parsing templates: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Step 5: Execute the template
	data := templateData{
		AllSports: sports,
		AllVenues: venues,
		AllTeams:  teams,
	}
	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("ERROR executing template: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (app *application) sportEventCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	data := models.InsertEventData{
		EventName:   r.PostForm.Get("eventName"),
		Description: r.PostForm.Get("description"),
	}

	data.SportID, err = strconv.Atoi(r.PostForm.Get("sportID"))
	if err != nil {
		http.Error(w, "Invalid value for Sport ID.", http.StatusBadRequest)
		return
	}

	data.VenueID, err = strconv.Atoi(r.PostForm.Get("venueID"))
	if err != nil {
		http.Error(w, "Invalid value for Venue ID.", http.StatusBadRequest)
		return
	}

	data.Team1ID, err = strconv.Atoi(r.PostForm.Get("team1ID"))
	if err != nil {
		http.Error(w, "Invalid value for Team 1 ID.", http.StatusBadRequest)
		return
	}

	data.Team2ID, err = strconv.Atoi(r.PostForm.Get("team2ID"))
	if err != nil {
		http.Error(w, "Invalid value for Team 2 ID.", http.StatusBadRequest)
		return
	}

	data.Team1Score, err = strconv.Atoi(r.PostForm.Get("team1Score"))
	if err != nil {
		http.Error(w, "Invalid value for Team 1 Score.", http.StatusBadRequest)
		return
	}

	data.Team2Score, err = strconv.Atoi(r.PostForm.Get("team2Score"))
	if err != nil {
		http.Error(w, "Invalid value for Team 2 Score.", http.StatusBadRequest)
		return
	}

	// Validation: Check if the same team was selected twice.
	if data.Team1ID == data.Team2ID {
		http.Error(w, "You cannot select the same team for both Team 1 and Team 2.", http.StatusBadRequest)
		return
	}

	// Date/Time Parsing
	eventDateTimeStr := r.PostForm.Get("eventDateTime")
	layout := "2006-01-02T15:04"
	data.EventDateTime, err = time.Parse(layout, eventDateTimeStr)
	if err != nil {
		http.Error(w, "Invalid Date format. Please use the date picker.", http.StatusBadRequest)
		return
	}

	// Database Insertion
	id, err := app.sportEvents.Insert(data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Redirect on Success
	redirectURL := fmt.Sprintf("/event/view/%d", id)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (app *application) sportEventDeletePost(w http.ResponseWriter, r *http.Request) {
	// 1. Get the ID from the URL, just like in the view handler.
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		http.NotFound(w, r)
		return
	}

	// 2. Call the new Delete() method on the model.
	err = app.sportEvents.Delete(id)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// 3. On success, redirect the user to the homepage.
	// We can't redirect back to the view page because it no longer exists.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
