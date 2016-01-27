package notes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/husio/x/auth"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/web"

	"golang.org/x/net/context"
)

func HandleCreateNote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		web.JSONErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	acc, ok := auth.AuthRequired(pg.DB(ctx), w, r)
	if !ok {
		return
	}

	if errs := validateNote(&note); len(errs) > 0 {
		web.JSONErrs(w, errs, http.StatusBadRequest)
		return
	}

	note.OwnerID = acc.AccountID

	n, err := CreateNote(pg.DB(ctx), note)
	if err != nil {
		log.Printf("cannot create note: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	web.JSONResp(w, n, http.StatusCreated)
}

func validateNote(n *Note) []string {
	var errs []string
	if len(n.Content) < 3 {
		errs = append(errs, "'Content' must be at least 3 bytes long")
	}
	if len(n.Content) > 20000 {
		errs = append(errs, "'Content' too long")
	}
	if n.ExpireAt != nil {
		if n.ExpireAt.Before(time.Now()) {
			errs = append(errs, "if provided, 'ExpireAt' must be in the future")
		}
		if n.ExpireAt.After(time.Now().Add(300 * 24 * time.Hour)) {
			errs = append(errs, "'ExpireAt' must be less than 300 days in the future")
		}
	}
	return errs
}

func HandleNoteDetails(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	args := web.Args(ctx)
	note, err := NoteByID(pg.DB(ctx), stoint(args.ByIndex(0)))
	if err != nil {
		if err == pg.ErrNotFound {
			web.StdJSONErr(w, http.StatusNotFound)
		} else {
			log.Printf("cannot get %q note: %s", args.ByIndex(0), err)
			web.StdJSONErr(w, http.StatusInternalServerError)
		}
		return
	}

	if !note.IsPublic {
		acc, ok := auth.Authenticated(pg.DB(ctx), r)
		if !ok || acc.AccountID != note.OwnerID {
			web.StdJSONErr(w, http.StatusUnauthorized)
			return
		}
	}
	web.JSONResp(w, note, http.StatusOK)
}

func stoint(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func HandleUpdateNote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var input Note
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		web.JSONErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errs := validateNote(&input); len(errs) > 0 {
		web.JSONErrs(w, errs, http.StatusBadRequest)
		return
	}

	tx, err := pg.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot start transaction: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	acc, ok := auth.AuthRequired(tx, w, r)
	if !ok {
		return
	}
	noteID := stoint(web.Args(ctx).ByIndex(0))

	if ok, err := IsNoteOwner(tx, noteID, acc.AccountID); err != nil {
		log.Printf("cannot check %d note owner: %s", noteID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	} else if !ok {
		web.JSONErr(w, "you are not owner of this note", http.StatusUnauthorized)
		return
	}

	note, err := UpdateNote(tx, input)
	if err != nil {
		log.Printf("cannot update %d note: %s", noteID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	web.JSONResp(w, note, http.StatusOK)
}

func HandleDeleteNote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tx, err := pg.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot start transaction: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	acc, ok := auth.AuthRequired(tx, w, r)
	if !ok {
		return
	}
	noteID := stoint(web.Args(ctx).ByIndex(0))

	if ok, err := IsNoteOwner(tx, noteID, acc.AccountID); err != nil {
		if err == pg.ErrNotFound {
			web.StdJSONErr(w, http.StatusNotFound)
		} else {
			log.Printf("cannot check %d note owner: %s", noteID, err)
			web.StdJSONErr(w, http.StatusInternalServerError)
		}
		return
	} else if !ok {
		web.JSONErr(w, "you are not owner of this note", http.StatusUnauthorized)
		return
	}

	if err := DeleteNote(tx, noteID); err != nil {
		log.Printf("cannot delete %d note: %s", noteID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusGone)
}

func HandleListNotes(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := pg.DB(ctx)

	acc, ok := auth.AuthRequired(db, w, r)
	if !ok {
		return
	}

	var offset int
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err != nil {
			web.JSONErr(w, "invalid 'offset' value", http.StatusBadRequest)
			return
		} else {
			offset = n
		}
	}

	notes, err := NotesByOwner(db, acc.AccountID, 300, offset)
	if err != nil {
		log.Printf("cannot fetch notes for %d: %s", acc.AccountID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	if notes == nil {
		notes = make([]*Note, 0) // JSON api should return empty list
	}

	content := struct {
		Notes []*Note
	}{
		Notes: notes,
	}
	web.JSONResp(w, content, http.StatusOK)
}
