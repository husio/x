package notes

import (
	"time"

	"github.com/husio/x/storage/pg"
)

type Note struct {
	NoteID    int        `db:"note_id"`
	OwnerID   int        `db:"owner_id"`
	Content   string     `db:"content"`
	CreatedAt time.Time  `db:"created_at"`
	IsPublic  bool       `db:"is_public"`
	ExpireAt  *time.Time `db:"expire_at"`
}

func CreateNote(g pg.Getter, n Note) (*Note, error) {
	n.CreatedAt = time.Now()
	err := g.Get(&n.NoteID, `
		INSERT INTO notes (owner_id, content, expire_at, created_at, is_public)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING note_id
	`, n.OwnerID, n.Content, n.ExpireAt, time.Now(), n.IsPublic)
	return &n, pg.CastErr(err)
}

func UpdateNote(g pg.Getter, note Note) (*Note, error) {
	err := g.Get(&note, `
		UPDATE notes
		SET content = $1, is_public = $2, expire_at = $3
		WHERE note_id = $4
		RETURNING *
	`, note.Content, note.IsPublic, note.ExpireAt, note.NoteID)
	return &note, pg.CastErr(err)
}

func IsNoteOwner(g pg.Getter, noteID, ownerID int) (bool, error) {
	var isOwner bool
	err := g.Get(&isOwner, `
		SELECT COALESCE(sub.is_owner, false) FROM (
			SELECT true AS is_owner FROM notes
			WHERE note_id = $1 AND owner_id = $2
			LIMIT 1
		) sub
	`, noteID, ownerID)
	return isOwner, pg.CastErr(err)
}

func NoteByID(g pg.Getter, noteID int) (*Note, error) {
	var n Note
	err := g.Get(&n, `
		SELECT * FROM notes
		WHERE note_id = $1 AND expire_at < $2
		LIMIT 1
	`, noteID, time.Now())
	return &n, pg.CastErr(err)
}

func NotesByOwner(s pg.Selector, ownerID, limit, offset int) ([]*Note, error) {
	var notes []*Note
	err := s.Select(&notes, `
		SELECT * FROM notes
		WHERE owner_id = $1 AND (expire_at < $2 OR expire_at IS NULL)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, ownerID, time.Now(), limit, offset)
	return notes, pg.CastErr(err)
}

func DeleteNote(e pg.Execer, noteID int) error {
	res, err := e.Exec(`
		DELETE FROM notes WHERE note_id = $1
	`, noteID)
	if err != nil {
		return pg.CastErr(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return pg.ErrNotFound
	}
	return nil
}
