package pgtest

import (
	"database/sql"
	"errors"
	"reflect"

	"golang.org/x/net/context"

	"github.com/husio/x/storage/pg"
)

func WithDB(ctx context.Context, db *DB) context.Context {
	return context.WithValue(ctx, "storage.pg:db", db)
}

type DB struct {
	Stack  []ResultMock
	Fatalf func(string, ...interface{})
}

type ResultMock struct {
	Method string
	Result interface{}
	Err    error
}

func (db *DB) Beginx() (pg.Connection, error) {
	return db, nil
}

func (db *DB) Commit() error {
	return nil
}

func (db *DB) Rollback() error {
	return nil
}

func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	mock := db.pop("Get", query, args)
	if want, got := "Get", mock.Method; want != got {
		db.Fatalf("want %q, got %q call", want, got)
		return ErrUnexpectedCall
	}
	if mock.Result != nil {
		copyData(dest, mock.Result)
	}
	return mock.Err
}

func (db *DB) Select(dest interface{}, query string, args ...interface{}) error {
	mock := db.pop("Select", query, args)
	if want, got := "Select", mock.Method; want != got {
		db.Fatalf("want %q, got %q call", want, got)
		return ErrUnexpectedCall
	}
	if mock.Result != nil {
		copyData(dest, mock.Result)
	}
	return mock.Err
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	mock := db.pop("Exec", query, args)
	if want, got := "Exec", mock.Method; want != got {
		db.Fatalf("want %q, got %q call", want, got)
		return nil, ErrUnexpectedCall
	}
	res, ok := mock.Result.(sql.Result)
	if !ok {
		db.Fatalf("'Exec' result must be sql.Result type, got %T instead", mock.Result)
	}
	return res, mock.Err
}

func (db *DB) pop(method, query string, args []interface{}) *ResultMock {
	if len(db.Stack) == 0 {
		db.Fatalf("mock call stack empty: %q %s %+v", method, query, args)
		return nil
	}
	mock := db.Stack[0]
	db.Stack = db.Stack[1:]
	return &mock
}

func copyData(dst, src interface{}) {
	s := reflect.ValueOf(src)
	switch s.Kind() {
	case reflect.Ptr:
		starX := s.Elem()
		y := reflect.New(starX.Type())
		starY := y.Elem()
		starY.Set(starX)
		reflect.ValueOf(dst).Elem().Set(y.Elem())
	case reflect.Slice:
		panic("cannot copy data from slice, pass address instead")
	default:
		dst = s.Interface()
	}
}

var ErrUnexpectedCall = errors.New("unexpected call")
