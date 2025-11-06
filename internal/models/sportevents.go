package models

import (
	"database/sql"
	"errors"
	"time"
)

type EventHomeView struct {
	EventID       int
	EventDateTime time.Time
	VenueName     string
	SportName     string
	Team1Name     string
	Team1Score    int
	Team2Name     string
	Team2Score    int
}

type Team struct {
	ID          int
	Name        string
	Coach       string
	YearFounded int
}

// Represents a row from the 'Players' table
type Player struct {
	ID        int
	FirstName string
	LastName  string
	Position  string
}

// This is a special composite struct. It doesn't represent a single table.
// It holds all the information for one team's participation in an event.
type TeamInEvent struct {
	Team    Team
	Score   int
	Players []Player
}

type Sport struct {
	ID          int
	Name        string
	Description string
}

type Venue struct {
	ID       int
	Name     string
	Address  string
	City     string
	Country  string
	Capacity int
}

type SportEvent struct {
	EventID       int
	EventName     string
	EventDateTime time.Time
	Description   string
	SportID       int
	VenueID       int
}

type InsertEventData struct {
	EventName     string
	EventDateTime time.Time
	Description   string
	SportID       int
	VenueID       int
	Team1ID       int
	Team1Score    int
	Team2ID       int
	Team2Score    int
}

type SportEventModel struct {
	DB *sql.DB
}

func (m *SportEventModel) Insert(data InsertEventData) (int, error) {
	// 1. Begin a new transaction
	tx, err := m.DB.Begin()
	if err != nil {
		return 0, err
	}

	// 2. Insert into the 'events' table
	stmt := `INSERT INTO events (event_name, event_date_time, description, _sport_id, _venue_id)
         	 VALUES(?, ?, ?, ?, ?)`
	result, err := tx.Exec(stmt, data.EventName, data.EventDateTime, data.Description, data.SportID, data.VenueID)
	if err != nil {
		// If anything goes wrong, roll back the transaction and return the error.
		tx.Rollback()
		return 0, err
	}
	eventID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 3. Insert the two teams into the 'event_teams' join table
	stmt = `INSERT INTO event_teams (_event_id, _team_id, score) VALUES (?, ?, ?)`

	// Insert Team 1
	_, err = tx.Exec(stmt, eventID, data.Team1ID, data.Team1Score)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Insert Team 2
	_, err = tx.Exec(stmt, eventID, data.Team2ID, data.Team2Score)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 4. If everything was successful, commit the transaction.
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(eventID), nil
}

func (m *SportEventModel) Get(id int) (SportEvent, Sport, Venue, []TeamInEvent, error) {
	// === QUERY 1: Get Event, Sport, and Venue Details ===
	// This part is the same as before.
	stmt := `SELECT e.event_id, e.event_name, e.event_date_time, e.description,
                s.sport_id, s.sport_name, s.description,
                v.venue_id, v.venue_name, v.address, v.city, v.country, v.capacity
             FROM events AS e
             INNER JOIN sports AS s ON e._sport_id = s.sport_id
             INNER JOIN venues AS v ON e._venue_id = v.venue_id
             WHERE e.event_id = ?`

	row := m.DB.QueryRow(stmt, id)
	var e SportEvent
	var s Sport
	var v Venue
	var dateTimeString string
	err := row.Scan(&e.EventID, &e.EventName, &dateTimeString, &e.Description,
		&s.ID, &s.Name, &s.Description,
		&v.ID, &v.Name, &v.Address, &v.City, &v.Country, &v.Capacity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SportEvent{}, Sport{}, Venue{}, nil, ErrNoRecord
		} else {
			return SportEvent{}, Sport{}, Venue{}, nil, err
		}
	}
	parsedTime, err := parseFlexibleTime(dateTimeString)
	if err != nil {
		return SportEvent{}, Sport{}, Venue{}, nil, err
	}
	e.EventDateTime = parsedTime

	// === QUERY 2: Get all Teams and their Scores for this Event ===
	stmt = `SELECT t.team_id, t.team_name, t.coach, t.year_founded, et.score
            FROM teams AS t
            INNER JOIN event_teams AS et ON t.team_id = et._team_id
            WHERE et._event_id = ?`

	rows, err := m.DB.Query(stmt, id)
	if err != nil {
		return SportEvent{}, Sport{}, Venue{}, nil, err
	}
	defer rows.Close()

	var teamsInEvent []TeamInEvent
	for rows.Next() {
		var tie TeamInEvent // A single "Team In Event"
		err := rows.Scan(&tie.Team.ID, &tie.Team.Name, &tie.Team.Coach, &tie.Team.YearFounded, &tie.Score)
		if err != nil {
			return SportEvent{}, Sport{}, Venue{}, nil, err
		}
		teamsInEvent = append(teamsInEvent, tie)
	}
	if err = rows.Err(); err != nil {
		return SportEvent{}, Sport{}, Venue{}, nil, err
	}

	// === QUERY 3: Get Players for EACH Team ===
	// Now we loop through the teams we just found and fetch their players.
	for i := range teamsInEvent {
		stmt = `SELECT player_id, first_name, last_name, position FROM players WHERE _team_id = ?`
		playerRows, err := m.DB.Query(stmt, teamsInEvent[i].Team.ID)
		if err != nil {
			return SportEvent{}, Sport{}, Venue{}, nil, err
		}

		var players []Player
		for playerRows.Next() {
			var p Player
			err := playerRows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Position)
			if err != nil {
				playerRows.Close()
				return SportEvent{}, Sport{}, Venue{}, nil, err
			}
			players = append(players, p)
		}
		playerRows.Close() // Close each player result set before the next loop iteration.

		// Add the slice of players to the correct team.
		teamsInEvent[i].Players = players
	}

	// Finally, return all the rich data we've assembled.
	return e, s, v, teamsInEvent, nil
}

func (m *SportEventModel) Latest() ([]EventHomeView, error) {
	stmt := `
        SELECT
            e.event_id,
            e.event_date_time,
            v.venue_name,
            s.sport_name, -- <-- 1. ADDED THE SPORT NAME TO THE SELECT
            matchup.team1_name,
            matchup.team1_score,
            matchup.team2_name,
            matchup.team2_score
        FROM
            events AS e
        JOIN
            venues AS v ON e._venue_id = v.venue_id
        JOIN                                     -- <-- 2. ADDED THE JOIN TO THE SPORTS TABLE
            sports AS s ON e._sport_id = s.sport_id
        LEFT JOIN (
            -- The subquery for teams remains exactly the same
            SELECT
                et1._event_id, t1.team_name AS team1_name, et1.score AS team1_score,
                t2.team_name AS team2_name, et2.score AS team2_score
            FROM
                event_teams et1
            JOIN event_teams et2 ON et1._event_id = et2._event_id AND et1._team_id < et2._team_id
            JOIN teams t1 ON et1._team_id = t1.team_id
            JOIN teams t2 ON et2._team_id = t2.team_id
        ) AS matchup ON e.event_id = matchup._event_id
        ORDER BY
            e.event_date_time DESC
        LIMIT 10`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventHomeView
	for rows.Next() {
		var ehv EventHomeView
		var dateTimeString string
		var team1Name, team2Name sql.NullString
		var team1Score, team2Score sql.NullInt64

		err := rows.Scan(
			&ehv.EventID,
			&dateTimeString,
			&ehv.VenueName,
			&ehv.SportName, // <-- 3. ADDED THE SPORT NAME TO THE SCAN CALL (IN THE CORRECT POSITION)
			&team1Name,
			&team1Score,
			&team2Name,
			&team2Score,
		)
		if err != nil {
			return nil, err
		}

		ehv.Team1Name = team1Name.String
		ehv.Team1Score = int(team1Score.Int64)
		ehv.Team2Name = team2Name.String
		ehv.Team2Score = int(team2Score.Int64)

		parsedTime, err := parseFlexibleTime(dateTimeString)
		if err != nil {
			return nil, err
		}
		ehv.EventDateTime = parsedTime
		events = append(events, ehv)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (m *SportEventModel) GetAllSports() ([]Sport, error) {
	stmt := `SELECT sport_id, sport_name, description FROM sports ORDER BY sport_name`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sports []Sport
	for rows.Next() {
		var s Sport
		if err := rows.Scan(&s.ID, &s.Name, &s.Description); err != nil {
			return nil, err
		}
		sports = append(sports, s)
	}
	return sports, nil
}

func (m *SportEventModel) GetAllVenues() ([]Venue, error) {
	// THIS IS THE FIX: The SELECT statement now fetches all 6 columns.
	stmt := `SELECT venue_id, venue_name, address, city, country, capacity FROM venues ORDER BY venue_name`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var venues []Venue
	for rows.Next() {
		var v Venue
		// This Scan call is now correct because the number of columns from the SELECT matches.
		if err := rows.Scan(&v.ID, &v.Name, &v.Address, &v.City, &v.Country, &v.Capacity); err != nil {
			return nil, err
		}
		venues = append(venues, v)
	}
	return venues, nil
}

func (m *SportEventModel) GetAllTeams() ([]Team, error) {
	// THIS IS THE FIX: The SELECT statement now fetches all 4 columns
	// that your 'Team' struct requires.
	stmt := `SELECT team_id, team_name, coach, year_founded FROM teams ORDER BY team_name`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var t Team
		// This Scan call is now correct because it expects 4 values,
		// and the SELECT statement above provides exactly 4 columns.
		if err := rows.Scan(&t.ID, &t.Name, &t.Coach, &t.YearFounded); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (m *SportEventModel) Delete(id int) error {
	// 1. Begin a new transaction.
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}

	// 2. Delete the records from the 'event_teams' join table first.
	// This is important to respect foreign key constraints.
	stmt := `DELETE FROM event_teams WHERE _event_id = ?`
	_, err = tx.Exec(stmt, id)
	if err != nil {
		tx.Rollback() // Roll back on error
		return err
	}

	// 3. Delete the record from the 'events' table.
	stmt = `DELETE FROM events WHERE event_id = ?`
	_, err = tx.Exec(stmt, id)
	if err != nil {
		tx.Rollback() // Roll back on error
		return err
	}

	// 4. If both deletes were successful, commit the transaction.
	return tx.Commit()
}

func parseFlexibleTime(dateTimeString string) (time.Time, error) {
	// Layout 1: For dates WITH a timezone (like the ones from our form)
	layoutWithTz := "2006-01-02 15:04:05-07:00"
	parsedTime, err := time.Parse(layoutWithTz, dateTimeString)
	if err == nil {
		// If it works, we're done. Return the result.
		return parsedTime, nil
	}

	// Layout 2: For dates WITHOUT a timezone (like our original data)
	layoutWithoutTz := "2006-01-02 15:04:05"
	parsedTime, err = time.Parse(layoutWithoutTz, dateTimeString)
	if err == nil {
		// If this one works, return the result.
		return parsedTime, nil
	}

	// If neither layout worked, return the last error.
	return time.Time{}, err
}
