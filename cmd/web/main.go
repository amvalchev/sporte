package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Print("starting server on :8000")

	db, err := sql.Open("sqlite3", "sports_calendar.db")
	if err != nil {
		fmt.Print(err)
		return
	}

	defer db.Close()
	fmt.Println("Connected to the SQLite database successfully.")

	err = http.ListenAndServe(":8000", routes())
	log.Fatal(err)
}
