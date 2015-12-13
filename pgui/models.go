package main

import (
	"database/sql/driver"
	"time"
)

type StoredQuery struct {
	QueryID int64
	Name    string
	Created time.Time
	Starred bool
	Query   string
}

type StoredResult struct {
	ResultID  int64
	QueryID   int64
	RowCount  int
	Path      string
	WorkStart time.Time
	WorkEnd   NullTime
}

type NullTime struct {
	time.Time
	Valid bool
}

func (nt *NullTime) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}

func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

type ResultWithQuery struct {
	Result *StoredResult
	Query  *StoredQuery
}
