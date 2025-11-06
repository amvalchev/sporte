package main

import (
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
	// This handler is correct and does not need changes.
	files := []string{
		"./ui/html/base.tmpl", "./ui/html/partials/nav.tmpl", "./ui/html/pages/home.tmpl",
	}
	ts, err := template.ParseFiles(files...)
	if err != nil { /* ... */
	}
	events, err := app.sportEvents.Latest()
	if err != nil { /* ... */
	}
	data := templateData{HomeViewEvents: events}
	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil { /* ... */
	}
}

func (app *application) sportEventView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 { /* ... */
	}

	event, sport, venue, teams, err := app.sportEvents.Get(id)
	if err != nil { /* ... */
	}

	files := []string{
		"./ui/html/base.tmpl", "./ui/html/partials/nav.tmpl", "./ui/html/pages/view.tmpl",
	}
	ts, err := template.ParseFiles(files...)
	if err != nil { /* ... */
	}

	// FIX #1: We now put the data into the correct field: 'TeamsInEvent'.
	data := templateData{
		Event:        event,
		Sport:        sport,
		Venue:        venue,
		TeamsInEvent: teams,
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil { /* ... */
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

// FIX #2: The entire sportEventCreatePost handler is rewritten.
func (app *application) sportEventCreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// Create the struct we need to pass to the Insert method.
	data := models.InsertEventData{}

	// Parse and convert all the form fields, handling errors for each.
	data.EventName = r.PostForm.Get("eventName")
	data.Description = r.PostForm.Get("description")

	data.SportID, err = strconv.Atoi(r.PostForm.Get("sportID"))
	if err != nil {
		http.Error(w, "Invalid Sport ID", http.StatusBadRequest)
		return
	}

	data.VenueID, err = strconv.Atoi(r.PostForm.Get("venueID"))
	if err != nil {
		http.Error(w, "Invalid Venue ID", http.StatusBadRequest)
		return
	}

	data.Team1ID, err = strconv.Atoi(r.PostForm.Get("team1ID"))
	if err != nil {
		http.Error(w, "Invalid Team 1 ID", http.StatusBadRequest)
		return
	}

	data.Team2ID, err = strconv.Atoi(r.PostForm.Get("team2ID"))
	if err != nil {
		http.Error(w, "Invalid Team 2 ID", http.StatusBadRequest)
		return
	}

	data.Team1Score, err = strconv.Atoi(r.PostForm.Get("team1Score"))
	if err != nil {
		http.Error(w, "Invalid Team 1 Score", http.StatusBadRequest)
		return
	}

	data.Team2Score, err = strconv.Atoi(r.PostForm.Get("team2Score"))
	if err != nil {
		http.Error(w, "Invalid Team 2 Score", http.StatusBadRequest)
		return
	}

	eventDateTimeStr := r.PostForm.Get("eventDateTime")
	layout := "2006-01-02T15:04"
	data.EventDateTime, err = time.Parse(layout, eventDateTimeStr)
	if err != nil {
		http.Error(w, "Invalid Date Format", http.StatusBadRequest)
		return
	}

	// Now call the Insert method with the single 'data' struct.
	id, err := app.sportEvents.Insert(data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

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
