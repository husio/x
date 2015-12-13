package main

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sqlx.Connect("postgres", "user=piotr password=piotr dbname=pgui sslmode=disable")
	if err != nil {
		log.Fatalf("cannot connect to database: %s", err)
	}

	ui := NewPgUI(db)
	http.Handle("/", ui)

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("HTTP server error: %s", err)
	}
}
