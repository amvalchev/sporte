package main

import (
	"log"
	"net/http"
)

func main() {

	log.Print("starting server on :8000")

	err := http.ListenAndServe(":8000", routes())
	log.Fatal(err)
}
