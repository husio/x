package web

import (
	"net/http"

	"golang.org/x/net/context"
)

type application struct {
	rt  *Router
	ctx context.Context
}

func NewApplication(ctx context.Context, rt *Router) http.Handler {
	return &application{
		ctx: ctx,
		rt:  rt,
	}
}

func (app *application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.rt.ServeCtxHTTP(app.ctx, w, r)
}
