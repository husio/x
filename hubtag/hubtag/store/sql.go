package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"golang.org/x/net/context"
)

type Getter interface {
	Get(dest interface{}, query string, args ...interface{}) error
}

type Selector interface {
	Select(dest interface{}, query string, args ...interface{}) error
}

type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

const dbCtxName = "hubtag:store"

func WithStore(ctx context.Context, dbc *sql.DB) context.Context {
	dbx := sqlx.NewDb(dbc, "postgres") // only postgres dialect is supported
	return context.WithValue(ctx, dbCtxName, dbx)
}

func DB(ctx context.Context) *sqlx.DB {
	return ctx.Value(dbCtxName).(*sqlx.DB)
}

func AccountByID(g Getter, id int) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT * FROM accounts WHERE id = $1
	`, id)
	return &a, casterr(err)
}
func AccountByLogin(g Getter, login string) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT * FROM accounts WHERE login = $1
	`, login)
	return &a, casterr(err)
}

func CreateAccount(e Execer, githubID int, login string) (*Account, error) {
	now := time.Now()
	_, err := e.Exec(`
		INSERT INTO accounts (id, login, created)
		VALUES ($1, $2, $3)
	`, githubID, login, now)
	if err != nil {
		return nil, err
	}
	a := Account{
		ID:      githubID,
		Login:   login,
		Created: now,
	}
	return &a, nil
}

func AddVote(g Getter, entity string, account int) (*Vote, error) {
	var v Vote
	err := g.Get(&v, `
		INSERT INTO votes (entity, account, created)
		VALUES ($1, $2, $3)
		RETURNING *
	`, entity, account, time.Now())
	return &v, casterr(err)
}

func DelVote(e Execer, entity string, account int) error {
	res, err := e.Exec(`
		DELETE FROM votes WHERE entity = $1 AND account = $2
	`, entity, account)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n != 1 {
		return ErrNotFound
	}
	return nil
}

func CreateEntity(g Getter, account int) (*Entity, error) {
	var e Entity
	err := g.Get(&e, `
		INSERT INTO entities (key, owner, created, votes)
		VALUES ($1, $2, $3, 0)
		RETURNING *
	`, randomStr(18), account, time.Now())
	return &e, casterr(err)
}

func EntityVotes(s Selector, entity string, limit, offset int) ([]*Vote, error) {
	res := make([]*Vote, 0)
	err := s.Select(&res, `
		SELECT * FROM votes WHERE entity = $1 LIMIT $2 OFFSET $3
	`, entity, limit, offset)
	return res, casterr(err)
}

func EntityVotesCount(g Getter, entity string) (int, error) {
	var cnt int
	err := g.Get(&cnt, `
		SELECT votes FROM entities WHERE entity = $1 LIMIT 1
	`, entity)
	return cnt, casterr(err)
}

func EntityByKey(g Getter, key string) (*Entity, error) {
	var e Entity
	err := g.Get(&e, `
		SELECT * FROM entities WHERE key = $1
	`, key)
	return &e, casterr(err)
}

func EntitiesByOwner(s Selector, owner int, limit, offset int) ([]*Entity, error) {
	var res []*Entity
	err := s.Select(&res, `
		SELECT * FROM entities WHERE owner = $1 LIMIT $2 OFFSET $3
	`, owner, limit, offset)
	return res, casterr(err)
}

func CreateSession(e Execer, account int) (string, error) {
	key := randomStr(22)
	_, err := e.Exec(`
		INSERT INTO sessions (key, account, created) VALUES ($1, $2, $3)
	`, key, account, time.Now())
	return key, err
}

func SessionAccount(g Getter, key string) (*Account, error) {
	var a Account
	err := g.Get(&a, `
		SELECT a.id, a.login, a.created
		FROM accounts a INNER JOIN sessions s ON s.account = a.id
		WHERE s.key = $1
	`, key)
	return &a, casterr(err)
}

func randomStr(length int) string {
	b := make([]byte, length)
	if n, err := rand.Read(b); err != nil || n != length {
		panic(fmt.Sprintf("cannot read random value: %s", err))
	}
	return base32.StdEncoding.EncodeToString(b)[:length]
}

func casterr(err error) error {
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
		return ErrConflict
	}
	return err
}
