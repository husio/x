package auth

import (
	"fmt"
	"net/http"

	"github.com/husio/x/storage/pg"
)

var LoginUrl = "/login"

func AuthRequired(
	g pg.Getter,
	w http.ResponseWriter,
	r *http.Request,
) (*AccountWithScopes, bool) {
	if acc, ok := Authenticated(g, r); ok {
		return acc, true
	}
	u := fmt.Sprintf("%s?next=%s", LoginUrl, r.URL.Path)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
	return nil, false
}
