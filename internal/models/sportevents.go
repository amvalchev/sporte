package models

import (
	"database/sql"
	"errors"
	"time"
)

type SportEvent struct {
	EventID       int
	EventName     string
	EventDateTime time.Time
	Description   string
	SportID       int
	VenueID       int
}

type SportEventModel struct {
	DB *sql.DB
}

func (m *SportEventModel) Insert(eventname string, evendatetime time.Time, description string, sportid int, venueid int) (int, error) {
	stmt := `INSERT INTO events (event_name, event_date_time, description, _sport_id, _venue_id)
         	 VALUES(?, ?, ?, ?, ?)`

	result, err := m.DB.Exec(stmt, eventname, evendatetime, description, sportid, venueid)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *SportEventModel) Get(id int) (SportEvent, error) {
	stmt := `SELECT event_id, event_name, event_date_time, description, _sport_id, _venue_id
        	 FROM events WHERE event_id = ?`
	row := m.DB.QueryRow(stmt, id)

	var e SportEvent
	var dateTimeString string
	err := row.Scan(&e.EventID, &e.EventName, &dateTimeString, &e.Description, &e.SportID, &e.VenueID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SportEvent{}, ErrNoRecord
		} else {
			return SportEvent{}, err
		}
	}

	// Use our robust helper function to parse the date.
	parsedTime, err := parseFlexibleTime(dateTimeString)
	if err != nil {
		return SportEvent{}, err
	}
	e.EventDateTime = parsedTime

	return e, nil
}

func (m *SportEventModel) Latest() ([]SportEvent, error) {
	stmt := `SELECT event_id, event_name, event_date_time, description, _sport_id, _venue_id
         	 FROM events ORDER BY event_date_time DESC LIMIT 10`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sportEvents []SportEvent
	for rows.Next() {
		var dateTimeString string
		var e SportEvent
		err = rows.Scan(&e.EventID, &e.EventName, &dateTimeString, &e.Description, &e.SportID, &e.VenueID)
		if err != nil {
			return nil, err
		}

		// Use our robust helper function here as well.
		parsedTime, err := parseFlexibleTime(dateTimeString)
		if err != nil {
			return nil, err
		}
		e.EventDateTime = parsedTime

		sportEvents = append(sportEvents, e)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return sportEvents, nil
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
