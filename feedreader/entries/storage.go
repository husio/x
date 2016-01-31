package entries

import (
	"crypto/sha1"
	"encoding/base64"
	"time"

	"github.com/husio/x/storage/pg"
)

func ListEntries(s pg.Selector, limit, offset int) ([]*Entry, error) {
	var res []*Entry
	err := s.Select(&res, `
		SELECT * FROM entries
		ORDER BY created DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	return res, pg.CastErr(err)
}

func AddEntry(ex pg.Execer, e *Entry) error {
	h := sha1.Sum([]byte(e.Content))
	e.ID = base64.StdEncoding.EncodeToString(h[:])
	if e.Created.IsZero() {
		e.Created = time.Now()
	}
	_, err := ex.Exec(`
		INSERT INTO entries (id, title, link, content, created)
		VALUES ($1, $2, $3, $4, $5)
	`, e.ID, e.Title, e.Link, e.Content, e.Created)
	return pg.CastErr(err)
}

func AddResource(ex pg.Execer, r *Resource) error {
	_, err := ex.Exec(`
		INSERT INTO resources (link, created)
		VALUES ($1, $2)
	`, r.Link, r.Created)
	return pg.CastErr(err)
}

func ListResources(s pg.Selector, limit, offset int) ([]*Resource, error) {
	var res []*Resource
	err := s.Select(&res, `
		SELECT * FROM resources
		ORDER BY created DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	return res, pg.CastErr(err)
}

type Entry struct {
	ID      string
	Title   string
	Link    string
	Content string
	Created time.Time
}

type Resource struct {
	Title   string
	Link    string
	Created time.Time
}
