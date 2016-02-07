package pg

import (
	"database/sql"
	"errors"
	"log"

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

func WithDB(ctx context.Context, dbc *sql.DB) context.Context {
	dbx := sqlx.NewDb(dbc, "postgres")
	x := &sqlxdb{dbx: dbx}
	return context.WithValue(ctx, "storage.pg:db", x)
}

type Database interface {
	Beginx() (Connection, error)
	Getter
	Selector
	Execer
}

type Connection interface {
	Getter
	Selector
	Execer
	Rollback() error
	Commit() error
}

func DB(ctx context.Context) Database {
	db := ctx.Value("storage.pg:db")
	if db == nil {
		log.Print("missing database in context")
	}
	return db.(Database)
}

func CastErr(err error) error {
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
		return ErrConflict
	}
	return err
}

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// sqlxdb wraps sqlx.DB structure and provides custom function notations that
// can be easily mocked
type sqlxdb struct {
	dbx *sqlx.DB
}

func (x *sqlxdb) Beginx() (Connection, error) {
	return x.dbx.Beginx()
}

func (x *sqlxdb) Get(dest interface{}, query string, args ...interface{}) error {
	return x.dbx.Get(dest, query, args...)
}

func (x *sqlxdb) Select(dest interface{}, query string, args ...interface{}) error {
	return x.dbx.Select(dest, query, args...)
}

func (x *sqlxdb) Exec(query string, args ...interface{}) (sql.Result, error) {
	return x.dbx.Exec(query, args...)
}
