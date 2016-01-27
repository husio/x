package auth

import (
	"net/http"

	"github.com/husio/x/storage/pg"
)

func AuthRequired(g pg.Getter, w http.ResponseWriter, r *http.Request) (*Account, bool) {
	if acc, ok := Authenticated(g, r); ok {
		return acc, true
	}
	u := "/login?next=" + r.URL.Path
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
	return nil, false
}
