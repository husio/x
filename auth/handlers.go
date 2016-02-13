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

	"github.com/husio/x/cache"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/web"
)

const stateCookie = "oauthState"

func WithOAuth(ctx context.Context, conf map[string]*oauth2.Config) context.Context {
	return context.WithValue(ctx, "auth:oauth", conf)
}

func oauth(ctx context.Context, name string) (*oauth2.Config, bool) {
	val := ctx.Value("auth:oauth")
	if val == nil {
		panic("oauth configuration not present in context")
	}
	conf, ok := val.(map[string]*oauth2.Config)
	return conf[name], ok
}

func LoginHandler(provider string) web.HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		conf, ok := oauth(ctx, provider)
		if !ok {
			log.Printf("missing oauth provider configuration: %s", provider)
			const code = http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			return
		}

		state := randStr(18)
		url := conf.AuthCodeURL(state, oauth2.AccessTypeOnline)
		http.SetCookie(w, &http.Cookie{
			Name:    stateCookie,
			Path:    "/",
			Value:   state,
			Expires: time.Now().Add(time.Minute * 15),
		})

		nextURL := r.URL.Query().Get("next")
		if nextURL == "" {
			nextURL = "/"
		}
		err := cache.Get(ctx).Put("auth:"+state, &authData{
			Provider: provider,
			Scopes:   conf.Scopes,
			NextURL:  nextURL,
		})
		if err != nil {
			log.Printf("cannot store in cache: %s", err)
			web.StdJSONErr(w, http.StatusInternalServerError)
			return
		}
		web.JSONRedirect(w, url, http.StatusTemporaryRedirect)
	}
}

type authData struct {
	Provider string
	Scopes   []string
	NextURL  string
}

func HandleLoginCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var state string
	if c, err := r.Cookie(stateCookie); err != nil || c.Value == "" {
		log.Printf("invalid oauth state: expected %q, got %q", state, r.FormValue("state"))
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	} else {
		state = c.Value
	}

	if r.FormValue("state") != state {
		log.Printf("invalid oauth state: expected %q, got %q", state, r.FormValue("state"))
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	var data authData
	switch err := cache.Get(ctx).Get("auth:"+state, &data); err {
	case nil:
		// all good
	case cache.ErrNotFound:
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	default:
		log.Printf("cannot get auth data from cache: %s", err)
		web.JSONRedirect(w, "/", http.StatusTemporaryRedirect)
		return
	}

	conf, ok := oauth(ctx, data.Provider)
	if !ok {
		log.Printf("missing oauth provider configuration: %#v", data)
		const code = http.StatusInternalServerError
		http.Error(w, http.StatusText(code), code)
		return
	}

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

	provider := strings.SplitN(data.Provider, ":", 2)[0]
	acc, err := AccountByLogin(tx, *user.Login, provider)
	if err != nil {
		if err != pg.ErrNotFound {
			log.Printf("cannot get account %s: %s", *user.Login, err)
			http.Error(w, "cannot authenticate", http.StatusInternalServerError)
			return
		}

		acc, err = CreateAccount(tx, *user.ID, *user.Login, provider)
		if err != nil {
			log.Printf("cannot create account for %v: %s", user, err)
			http.Error(w, "cannot create account", http.StatusInternalServerError)
			return
		}
	}

	if err := authenticate(tx, w, acc.AccountID, token.AccessToken, data.Scopes); err != nil {
		log.Printf("cannot authenticate %#v: %s", acc, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}

	web.JSONRedirect(w, data.NextURL, http.StatusTemporaryRedirect)
}

func authenticate(
	g pg.Getter,
	w http.ResponseWriter,
	account int,
	token string,
	scopes []string,
) error {
	key, err := CreateSession(g, account, randStr(48), token, scopes)
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
