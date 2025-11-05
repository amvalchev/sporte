package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/amvalchev/sporte/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

type application struct {
	sportEvents *models.SportEventModel
}

func main() {
	dsn := flag.String("dsn", "./sports_calendar.db", "SQLite3 Database DSN")
	flag.Parse()

	log.Print("starting server on :8000")

	db, err := openDB(*dsn)
	if err != nil {
		fmt.Print(err)
		return
	}

	defer db.Close()
	fmt.Println("Connected to the SQLite database successfully.")

	app := &application{
		sportEvents: &models.SportEventModel{DB: db},
	}

	err = http.ListenAndServe(":8000", app.routes())
	log.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn+"?_loc=auto")
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
