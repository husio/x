package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	oauth2gh "golang.org/x/oauth2/github"

	"github.com/husio/x/auth"
	"github.com/husio/x/envconf"
	"github.com/husio/x/paste/notes"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/web"
)

var router = web.NewRouter("", web.Routes{
	{"GET", `^/login$`, auth.HandleLoginGithub},
	{"GET", `^/login/github/success$`, auth.HandleLoginGithubCallback},

	{"GET   ", `^/api/v1/notes$`, notes.HandleListNotes},
	{"POST  ", `^/api/v1/notes$`, notes.HandleCreateNote},
	{"PUT   ", `^/api/v1/notes/{note-id:\d+}$`, notes.HandleUpdateNote},
	{"GET   ", `^/api/v1/notes/{note-id:\d+}$`, notes.HandleNoteDetails},
	{"DELETE", `^/api/v1/notes/{note-id:\d+}$`, notes.HandleDeleteNote},

	{"GET,POST,PUT,DELETE", `^/api/.*`, handleApi404},

	{"GET", `.*`, handleStaticDir},
})

var statics http.Handler

func handleStaticDir(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	statics.ServeHTTP(w, r)
}

func handleApi404(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	web.StdJSONErr(w, http.StatusNotFound)
}

func main() {
	log.SetFlags(log.Lshortfile)

	conf := envconf.Parse()

	httpAddr, _ := conf.String("HTTP", "localhost:8000", "HTTP server address")
	oauth := &oauth2.Config{
		ClientID:     conf.ReqString("GITHUB_KEY", "Github OAuth key"),
		ClientSecret: conf.ReqString("GITHUB_SECRET", "Github OAuth secret"),
		Scopes:       []string{},
		Endpoint:     oauth2gh.Endpoint,
	}

	dbname := conf.ReqString("DB_NAME", "Postgres database name")
	dbuser := conf.ReqString("DB_USER", "Postgres database user")
	dbpass := conf.ReqString("DB_PASS", "Postgres database password")

	staticDir := conf.ReqString("STATICS", "Static directory path ('./public')")
	statics = http.FileServer(http.Dir(staticDir))

	conf.Finish()

	ctx := context.Background()
	ctx = auth.WithGithubOAuth(ctx, oauth)

	db, err := sql.Open("postgres",
		fmt.Sprintf("dbname='%s' user='%s' password='%s' sslmode=disable", dbname, dbuser, dbpass))
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
	if err := http.ListenAndServe(httpAddr, app); err != nil {
		log.Printf("HTTP server error: %s", err)
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
