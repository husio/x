package store

import "time"

type Account struct {
	ID      int
	Login   string
	Created time.Time
}

type Vote struct {
	Entity  string
	Account int
	Created time.Time
}

type Entity struct {
	Key     string
	Owner   int
	Created time.Time
	Votes   int
}
