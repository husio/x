package pg

import (
	"database/sql"
	"errors"

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
	return context.WithValue(ctx, "storage.pg:db", dbx)
}

func DB(ctx context.Context) *sqlx.DB {
	return ctx.Value("storage.pg:db").(*sqlx.DB)
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
