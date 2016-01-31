package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/husio/x/storage/pg"
	"github.com/husio/x/web"
)

const state = "github-oauth" // TODO

func WithGithubOAuth(ctx context.Context, c *oauth2.Config) context.Context {
	return context.WithValue(ctx, "auth:github", c)
}

func githubOAuth(ctx context.Context) *oauth2.Config {
	return ctx.Value("auth:github").(*oauth2.Config)
}

func HandleLoginGithub(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	url := githubOAuth(ctx).AuthCodeURL(state, oauth2.AccessTypeOnline)
	if next := r.URL.Query().Get("next"); next != "" {
		http.SetCookie(w, &http.Cookie{
			Name:    nextCookieName,
			Path:    "/",
			Value:   next,
			Expires: time.Now().Add(time.Minute * 15),
		})
	}
	web.JSONRedirect(w, url, http.StatusTemporaryRedirect)
}

const nextCookieName = "loginNext"

func HandleLoginGithubCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.FormValue("state") != state {
		log.Printf("invalid oauth state: expected %q, got %q", state, r.FormValue("state"))
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	conf := githubOAuth(ctx)

	token, err := conf.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		log.Printf("oauth exchange failed: %s", err)
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}
	cli := github.NewClient(conf.Client(oauth2.NoContext, token))
	user, _, err := cli.Users.Get("")
	if err != nil {
		log.Printf("cannot get user: %s", err)
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	db := pg.DB(ctx)
	tx, err := db.Beginx()
	if err != nil {
		log.Printf("cannot start transaction: %s", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	acc, err := AccountByLogin(tx, *user.Login)
	if err != nil {
		if err != pg.ErrNotFound {
			log.Printf("cannot get account %s: %s", user.Name, err)
			http.Error(w, "cannot authenticate", http.StatusInternalServerError)
			return
		}

		acc, err = CreateAccount(tx, *user.ID, *user.Login)
		if err != nil {
			log.Printf("cannot create account for %v: %s", user, err)
			http.Error(w, "cannot create account", http.StatusInternalServerError)
			return
		}
	}

	if err := authenticate(tx, w, acc.AccountID, token.AccessToken); err != nil {
		log.Printf("cannot authenticate %#v: %s", acc, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	next := "/"
	if c, err := r.Cookie(nextCookieName); err == nil {
		next = c.Value
		http.SetCookie(w, &http.Cookie{
			Name:    nextCookieName,
			Path:    "/",
			Value:   "",
			Expires: time.Now().Add(-24 * time.Hour),
		})
	}
	web.JSONRedirect(w, next, http.StatusTemporaryRedirect)
}

func authenticate(e pg.Execer, w http.ResponseWriter, account int, token string) error {
	key, err := CreateSession(e, account, randStr(48), token)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:    userCookieName,
		Path:    "/",
		Value:   key,
		Expires: time.Now().Add(time.Hour * 24 * 7),
	})
	return nil
}

const userCookieName = "u"

func randStr(length int) string {
	b := make([]byte, length)
	if n, err := rand.Read(b); err != nil || n != length {
		panic(fmt.Sprintf("cannot read random value: %s", err))
	}
	s := base32.StdEncoding.EncodeToString(b)
	return strings.ToLower(s[:length])
}
