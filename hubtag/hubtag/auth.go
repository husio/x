package hubtag

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	oauth2gh "golang.org/x/oauth2/github"

	"github.com/husio/x/hubtag/hubtag/store"
	"github.com/husio/x/web"
)

var oauthConf = &oauth2.Config{
	ClientID:     os.Getenv("GITHUB_KEY"),
	ClientSecret: os.Getenv("GITHUB_SECRET"),
	Scopes:       []string{},
	Endpoint:     oauth2gh.Endpoint,
}

const state = "github-oauth" // TODO

func handleLoginGithub(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	url := oauthConf.AuthCodeURL(state, oauth2.AccessTypeOnline)
	web.JSONRedirect(w, url, http.StatusTemporaryRedirect)
}

func handleLoginGithubCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.FormValue("state") != state {
		log.Printf("invalid oauth state: expected %q, got %q", state, r.FormValue("state"))
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	token, err := oauthConf.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		log.Printf("oauth exchange failed: %s", err)
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	cli := github.NewClient(oauthConf.Client(oauth2.NoContext, token))
	user, _, err := cli.Users.Get("")
	if err != nil {
		log.Printf("cannot get user: %s", err)
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	db := store.DB(ctx)
	tx, err := db.Beginx()
	if err != nil {
		log.Printf("cannot start transaction: %s", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	acc, err := store.AccountByLogin(tx, *user.Login)
	if err != nil {
		if err != store.ErrNotFound {
			log.Printf("cannot get account %q: %s", user.Name, err)
			http.Error(w, "cannot authenticate", http.StatusInternalServerError)
			return
		}

		acc, err = store.CreateAccount(tx, *user.ID, *user.Login)
		if err != nil {
			log.Printf("cannot create account for %v: %s", user, err)
			http.Error(w, "cannot create account", http.StatusInternalServerError)
			return
		}
	}

	if err := authenticate(tx, w, acc.ID); err != nil {
		log.Printf("cannot authenticate %#v: %s", acc, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
}

const (
	userCookieName = "u"
)

func authenticate(e store.Execer, w http.ResponseWriter, account int) error {
	key, err := store.CreateSession(e, account)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:    userCookieName,
		Path:    "/",
		Value:   key,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	})
	return nil
}

func Authenticated(g store.Getter, r *http.Request) (*store.Account, bool) {
	var key string

	if user, _, ok := r.BasicAuth(); ok && user != "" {
		key = user
	} else if c, err := r.Cookie(userCookieName); err == nil && c.Value != "" {
		key = c.Value
	}
	if key == "" {
		return nil, false
	}

	a, err := store.SessionAccount(g, key)
	if err != nil {
		log.Printf("cannot get account for session %q: %s", key, err)
		return nil, false
	}
	return a, true
}
