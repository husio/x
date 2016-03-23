package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/husio/x/storage/pg"
)

type Account struct {
	AccountID int `db:"account_id"`
	Login     string
	Provider  string
	Created   time.Time
}

type AccountWithScopes struct {
	*Account
	Scopes string
}

func Authenticated(g pg.Getter, r *http.Request) (*AccountWithScopes, bool) {
	key := SessionKey(r)
	if key == "" {
		return nil, false
	}

	a, err := SessionAccount(g, key)
	if err != nil {
		return nil, false
	}
	return a, true
}

func SessionKey(r *http.Request) string {
	if key, _, ok := r.BasicAuth(); ok && key != "" {
		return key
	}
	if val := r.Header.Get("Authorization"); val != "" {
		return val
	}
	if c, err := r.Cookie(userCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

func AccountByID(g pg.Getter, accountID int) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT * FROM accounts WHERE account_id = $1
		LIMIT 1
	`, accountID)
	return &a, pg.CastErr(err)
}

func AccountByLogin(g pg.Getter, login, provider string) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT * FROM accounts WHERE login = $1 AND provider = $2
		LIMIT 1
	`, login, provider)
	return &a, pg.CastErr(err)
}

func CreateAccount(e pg.Execer, id int, login, provider string) (*Account, error) {
	now := time.Now()
	_, err := e.Exec(`
		INSERT INTO accounts (account_id, login, created, provider)
		VALUES ($1, $2, $3, $4)
	`, id, login, now, provider)
	if err != nil {
		return nil, err
	}
	a := Account{
		AccountID: id,
		Login:     login,
		Created:   now,
		Provider:  provider,
	}
	return &a, nil
}

func CreateSession(
	g pg.Getter,
	account int,
	key string,
	accessToken string,
	scopes []string,
) (string, error) {
	var ok bool
	err := g.Get(&ok, `
		INSERT INTO sessions (key, account, created, access_token, provider, scopes)
			SELECT $1, $2, $3, $4, a.provider, $5
			FROM accounts a WHERE a.account_id = $2 LIMIT 1
		RETURNING true
	`, key, account, time.Now(), accessToken, strings.Join(scopes, " "))
	if err != nil {
		return "", pg.CastErr(err)
	}
	if !ok {
		return "", pg.ErrNotFound
	}
	return key, nil
}

func SessionAccount(g pg.Getter, key string) (*AccountWithScopes, error) {
	var a AccountWithScopes
	err := g.Get(&a, `
		SELECT a.account_id, a.login, a.created, s.scopes
		FROM accounts a INNER JOIN sessions s ON s.account = a.account_id
		WHERE s.key = $1
		LIMIT 1
	`, key)
	return &a, pg.CastErr(err)
}

func AccessToken(g pg.Getter, accountID int) (string, error) {
	var token string
	err := g.Get(&token, `
		SELECT access_token FROM sessions
		WHERE account = $1
		ORDER BY created DESC
		LIMIT 1
	`, accountID)
	return token, pg.CastErr(err)
}
