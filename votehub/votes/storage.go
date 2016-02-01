package votes

import (
	"time"

	"github.com/husio/x/storage/pg"
)

type Vote struct {
	Entity  string
	Account int
	Created time.Time
}

type Counter struct {
	CounterID   int
	OwnerID     int
	Created     time.Time
	Description string
	Value       int
	URL         string
}

func AddVote(g pg.Getter, counterID, accountID int) (*Vote, error) {
	var v Vote
	err := g.Get(&v, `
		INSERT INTO votes (counter_id, account_id, created)
		VALUES ($1, $2, $3)
		RETURNING *
	`, counterID, accountID, time.Now())
	return &v, pg.CastErr(err)
}

func DelVote(e pg.Execer, counterID, accountID int) error {
	res, err := e.Exec(`
		DELETE FROM votes
		WHERE counter_id = $1 AND account_id = $2
	`, counterID, accountID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n != 1 {
		return pg.ErrNotFound
	}
	return nil
}

func CreateCounter(g pg.Getter, c Counter) (*Counter, error) {
	err := g.Get(&c, `
		INSERT INTO counters (value, owner, created, description, url)
		VALUES (0, $1, $2, $3, $4)
		RETURNING *
	`, 0, c.OwnerID, time.Now(), c.Description, c.URL)
	return &c, pg.CastErr(err)
}

func CounterVotes(s pg.Selector, counterID, limit, offset int) ([]*Vote, error) {
	res := make([]*Vote, 0)
	err := s.Select(&res, `
		SELECT * FROM votes WHERE counter_id = $1 LIMIT $2 OFFSET $3
	`, counterID, limit, offset)
	return res, pg.CastErr(err)
}

func CounterVotesCount(g pg.Getter, counterID int) (int, error) {
	var cnt int
	err := g.Get(&cnt, `
		SELECT votes FROM counters WHERE counter_id = $1 LIMIT 1
	`, counterID)
	return cnt, pg.CastErr(err)
}

func CounterByID(g pg.Getter, id int) (*Counter, error) {
	var c Counter
	err := g.Get(&c, `
		SELECT * FROM counters WHERE id = $1 LIMIT 1
	`, id)
	return &c, pg.CastErr(err)
}

func CountersByOwner(s pg.Selector, owner int, limit, offset int) ([]*Counter, error) {
	var res []*Counter
	err := s.Select(&res, `
		SELECT * FROM counters
		WHERE owner = $1
		ORDER BY created DESC
		LIMIT $2 OFFSET $3
	`, owner, limit, offset)
	return res, pg.CastErr(err)
}
