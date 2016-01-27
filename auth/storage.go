package auth

import (
	"log"
	"net/http"
	"time"

	"github.com/husio/x/storage/pg"
)

type Account struct {
	AccountID int `db:"account_id"`
	Login     string
	Created   time.Time
}

func Authenticated(g pg.Getter, r *http.Request) (*Account, bool) {
	key := SessionKey(r)
	if key == "" {
		return nil, false
	}

	a, err := SessionAccount(g, key)
	if err != nil {
		log.Printf("cannot get account for session %q: %s", key, err)
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
	`, accountID)
	return &a, pg.CastErr(err)
}
func AccountByLogin(g pg.Getter, login string) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT * FROM accounts WHERE login = $1
	`, login)
	return &a, pg.CastErr(err)
}

func CreateAccount(e pg.Execer, githubID int, login string) (*Account, error) {
	now := time.Now()
	_, err := e.Exec(`
		INSERT INTO accounts (account_id, login, created)
		VALUES ($1, $2, $3)
	`, githubID, login, now)
	if err != nil {
		return nil, err
	}
	a := Account{
		AccountID: githubID,
		Login:     login,
		Created:   now,
	}
	return &a, nil
}

func CreateSession(e pg.Execer, account int, key string) (string, error) {
	_, err := e.Exec(`
		INSERT INTO sessions (key, account, created) VALUES ($1, $2, $3)
	`, key, account, time.Now())
	return key, pg.CastErr(err)
}

func SessionAccount(g pg.Getter, key string) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT a.account_id, a.login, a.created
		FROM accounts a INNER JOIN sessions s ON s.account = a.account_id
		WHERE s.key = $1
	`, key)
	return &a, pg.CastErr(err)
}
