package hubtag

import (
	"database/sql"
	"net/http"

	"github.com/husio/x/hubtag/hubtag/store"
	"github.com/husio/x/web"

	"golang.org/x/net/context"
)

type App struct {
	rt  *web.Router
	ctx context.Context
}

func NewApp(dbc *sql.DB) *App {
	app := App{}

	ctx := context.Background()
	app.ctx = store.WithStore(ctx, dbc)

	app.rt = web.NewRouter("", web.Routes{
		{"GET", "/", handleMainPage},

		{"GET", "/login", handleLoginGithub},
		{"GET", "/login/github/success", handleLoginGithubCallback},

		{"GET", "/api/v1/entities", handleEntityList},
		{"POST", "/api/v1/entities", handleEntityCreate},
		{"GET", "/api/v1/entities/{entity-id}", handleEntityDetails},
		{"GET", "/api/v1/entities/{entity-id}/votes", handleEntityVotes},
		{"POST", "/api/v1/entities/{entity-id}/upvote", handleAddVote},
		{"DELETE", "/api/v1/entities/{entity-id}/upvote", handleDelVote},

		// both handlers must be accesable via GET
		{"GET", "/e/{entity-id}/banner.png", handleRenderBanner},
		{"GET", "/e/{entity-id}/upvote", handleAddVote},
		{"GET", "/e/{entity-id}/downvote", handleDelVote},

		{"GET,POST,PUT,DELETE", ".*", handle404},
	})
	return &app
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.rt.ServeCtxHTTP(app.ctx, w, r)
}

func handle404(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	web.StdJSONResp(w, http.StatusNotFound)
}
