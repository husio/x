package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/husio/x/feedreader/entries"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/web"
	"golang.org/x/net/context"
)

var router = web.NewRouter("", web.Routes{
	{"GET ", "^/$", entries.HandleListEntries},
	{"GET ", "^/resources$", entries.HandleListResources},
	{"POST", "^/resources$", entries.HandleAddResource},

	{"GET,POST,PUT,DELETE", ".*", handle404},
})

func main() {
	log.SetFlags(log.Lshortfile)

	ctx := context.Background()

	db, err := sql.Open("postgres", "")
	if err != nil {
		log.Fatalf("cannot connect to PostgreSQL: %s", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("cannot ping database: %s", err)
	}
	ctx = pg.WithDB(ctx, db)

	app := &application{
		ctx: ctx,
		rt:  router,
	}
	if err := http.ListenAndServe("localhost:8000", app); err != nil {
		log.Fatalf("HTTP error: %s", err)
	}
}

func handle404(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not found", http.StatusNotFound)
}

type application struct {
	ctx context.Context
	rt  *web.Router
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	app.rt.ServeCtxHTTP(app.ctx, w, r)
	workTime := time.Now().Sub(start) / time.Millisecond * time.Millisecond
	fmt.Printf(":: %5s %5s %s\n", workTime, r.Method, r.URL)
}
