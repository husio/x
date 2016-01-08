package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/husio/x/hubtag/hubtag"
	_ "github.com/lib/pq"
)

func main() {
	dbname := env("DB_NAME", "hubtag")
	dbuser := env("DB_USER", os.Getenv("USER"))
	dbpass := env("DB_PASS", dbuser)
	db, err := sql.Open("postgres",
		fmt.Sprintf("dbname=%s user=%s password=%s sslmode=disable", dbname, dbuser, dbpass))
	if err != nil {
		log.Fatalf("cannot connect to database: %s", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("cannot ping database: %s", err)
	}

	app := hubtag.NewApp(db)
	httpLn := env("HTTP", "localhost:8000")
	if err := http.ListenAndServe(httpLn, app); err != nil {
		log.Fatalf("cannot start HTTP server: %s", err)
	}
}

func env(name, def string) string {
	val := os.Getenv(name)
	if val == "" {
		return def
	}
	return def
}
