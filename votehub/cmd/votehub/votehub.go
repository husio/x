package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/husio/x/auth"
	"github.com/husio/x/envconf"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/cache"
	"github.com/husio/x/votehub/votes"
	"github.com/husio/x/votehub/webhooks"
	"github.com/husio/x/web"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	oauth2gh "golang.org/x/oauth2/github"
)

var router = web.NewRouter("", web.Routes{
	web.GET(`/login`, "login", auth.HandleLoginGithub),
	web.GET(`/login/github/success`, "login-github-callback", auth.HandleLoginGithubCallback),

	web.GET(`/`, "", votes.HandleListCounters),
	web.GET(`/counters`, "counters-listing", votes.HandleListCounters),

	web.GET(`/webhooks/create`, "webhooks-listing", webhooks.HandleListWebhooks),
	web.POST(`/webhooks/create`, "webhooks-create", webhooks.HandleCreateWebhooks),
	web.POST(`/webhooks/issues`, "webhooks-issues-callback", webhooks.HandleIssuesWebhookEvent),

	web.GET(`/v/{counter-id:\d+}/upvote`, "counters-upvote", votes.HandleClickUpvote),
	web.GET(`/v/{counter-id:\d+}/banner.svg`, "countes-banner-svg", votes.HandleRenderSVGBanner),

	web.ANY(`.*`, "", handle404),
})

func main() {
	conf := envconf.Parse()
	httpAddr, _ := conf.String("HTTP", "localhost:8000", "HTTP server address")
	oauth := &oauth2.Config{
		ClientID:     conf.ReqString("GITHUB_KEY", "Github OAuth key"),
		ClientSecret: conf.ReqString("GITHUB_SECRET", "Github OAuth secret"),
		Scopes:       []string{"public_repo", "write:repo_hook"},
		Endpoint:     oauth2gh.Endpoint,
	}
	dbname := conf.ReqString("DB_NAME", "Postgres database name")
	dbuser := conf.ReqString("DB_USER", "Postgres database user")
	dbpass := conf.ReqString("DB_PASS", "Postgres database password")
	conf.Finish()

	ctx := context.Background()
	ctx = auth.WithGithubOAuth(ctx, oauth)
	ctx = cache.WithIntCache(ctx)
	ctx = web.WithRouter(ctx, router)

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
	log.Printf("running HTTP server: %s", httpAddr)
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
