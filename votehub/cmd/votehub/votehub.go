package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/husio/x/auth"
	"github.com/husio/x/cache"
	"github.com/husio/x/envconf"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/core"
	"github.com/husio/x/votehub/ghub"
	"github.com/husio/x/votehub/votes"
	"github.com/husio/x/votehub/webhooks"
	"github.com/husio/x/web"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	oauth2gh "golang.org/x/oauth2/github"
)

var router = web.NewRouter("", web.Routes{
	web.GET(`/login`, "login", web.RedirectHandler("/login/basic", http.StatusSeeOther)),
	web.GET(`/login/basic`, "login-basic", auth.LoginHandler("github:basic")),
	web.GET(`/login/repo-owner`, "login-repo-owner", auth.LoginHandler("github:repo-owner")),
	web.GET(`/login/github/success`, "login-github-callback", auth.HandleLoginCallback),

	web.GET(`/`, "", votes.HandleListCounters),
	web.GET(`/counters`, "counters-listing", votes.HandleListCounters),

	web.GET(`/webhooks/create`, "webhooks-listing", webhooks.HandleListWebhooks),
	web.POST(`/webhooks/create`, "webhooks-create", webhooks.HandleCreateWebhooks),
	web.POST(`/webhooks/callbacks/issues`, "webhooks-issues-callback", webhooks.HandleIssuesWebhookCallback),

	web.GET(`/v/{counter-id:\d+}/upvote`, "counters-upvote", votes.HandleClickUpvote),
	web.GET(`/v/{counter-id:\d+}/banner.svg`, "counters-banner-svg", votes.HandleRenderSVGBanner),

	web.ANY(`.*`, "", handle404),
})

func main() {
	log.SetFlags(log.Lshortfile)

	conf := envconf.Parse()
	httpAddr, _ := conf.String("HTTP", "localhost:8000", "HTTP server address")
	oauthConf := map[string]*oauth2.Config{
		"github:basic": &oauth2.Config{
			ClientID:     conf.ReqString("GITHUB_KEY", "Github OAuth key"),
			ClientSecret: conf.ReqString("GITHUB_SECRET", "Github OAuth secret"),
			Scopes:       []string{},
			Endpoint:     oauth2gh.Endpoint,
		},
		"github:repo-owner": &oauth2.Config{
			ClientID:     conf.ReqString("GITHUB_KEY", "Github OAuth key"),
			ClientSecret: conf.ReqString("GITHUB_SECRET", "Github OAuth secret"),
			Scopes:       []string{"public_repo", "write:repo_hook"},
			Endpoint:     oauth2gh.Endpoint,
		},
	}
	dbconf := conf.ReqString("DB", "Postgres database connection")
	tmpls, _ := conf.String("TEMPLATES", "*/templates/*.html", "Glob path for HTML template files")
	tcache, _ := conf.Bool("TEMPLATES_CACHE", true, "If false, templates are not cached")
	conf.Finish()

	if err := core.LoadTemplates(tmpls, tcache); err != nil {
		log.Fatalf("cannot load templates: %s", err)
	}

	ctx := context.Background()
	ctx = auth.WithOAuth(ctx, oauthConf)
	ctx = ghub.WithClient(ctx, ghub.StandardClient)
	ctx = web.WithRouter(ctx, router)
	ctx = cache.WithLocalCache(ctx, 1000)

	db, err := sql.Open("postgres", dbconf)
	if err != nil {
		log.Fatalf("cannot connect to PostgreSQL: %s", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("cannot ping database: %s", err)
	}

	if err := core.LoadSchema(db); err != nil {
		log.Printf("cannot load schema: %s", err)
		if serr, ok := err.(*core.SchemaError); ok {
			log.Printf("query: %s", serr.Query)
		}
		os.Exit(1)
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
