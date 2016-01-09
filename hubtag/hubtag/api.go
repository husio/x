package hubtag

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/husio/x/hubtag/hubtag/store"
	"github.com/husio/x/web"

	"golang.org/x/net/context"
)

func handleMainPage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	a, _ := Authenticated(store.DB(ctx), r)
	var content = struct {
		Hello   string
		Account *store.Account
	}{
		Hello:   "World",
		Account: a,
	}
	web.JSONResp(w, content, http.StatusOK)
}

func handleEntityList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := store.DB(ctx)
	acc, ok := Authenticated(db, r)
	if !ok {
		web.StdJSONErr(w, http.StatusUnauthorized)
		return
	}

	off := 0
	if raw, ok := r.URL.Query()["offset"]; ok && len(raw) == 1 {
		if n, err := strconv.ParseInt(raw[0], 0, 32); err == nil {
			off = int(n)
		}
	}

	entities, err := store.EntitiesByOwner(db, acc.ID, 200, off)
	if err != nil {
		log.Printf("cannot get entities for %d: %s", acc.ID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	resp := struct {
		Entities []*store.Entity
	}{
		Entities: entities,
	}
	web.JSONResp(w, resp, http.StatusOK)
}

func handleEntityDetails(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	entityID := web.Args(ctx).ByIndex(0)
	db := store.DB(ctx)
	entity, err := store.EntityByKey(db, entityID)
	if err != nil {
		if err == store.ErrNotFound {
			text := fmt.Sprintf("Entity %q does not exist", entityID)
			web.JSONErr(w, text, http.StatusNotFound)
		} else {
			log.Printf("cannot get entity: %s", err)
			web.StdJSONErr(w, http.StatusInternalServerError)
		}
		return
	}

	web.JSONResp(w, entity, http.StatusOK)
}

func handleEntityVotes(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	entityID := web.Args(ctx).ByIndex(0)
	db := store.DB(ctx)
	const limit = 1000

	var offset int
	if raw := r.URL.Query().Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			web.JSONErr(w, "Invalid offset value", http.StatusBadRequest)
			return
		}
		offset = n
	}

	votes, err := store.EntityVotes(db, entityID, limit, offset)
	if err != nil {
		log.Printf("cannot get %q entity votes: %s", entityID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	total := len(votes)
	if len(votes) == limit {
		total, err = store.EntityVotesCount(db, entityID)
		if err != nil {
			log.Printf("cannot count %q entity votes: %s", entityID, err)
			web.StdJSONErr(w, http.StatusInternalServerError)
			return
		}
	}

	var content = struct {
		Votes []*store.Vote
		Total int
	}{
		Votes: votes,
		Total: total,
	}
	web.JSONResp(w, content, http.StatusOK)
}

func handleEntityCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tx, err := store.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot create transcation: %s", err)
		web.StdJSONErr(w, http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	account, ok := Authenticated(tx, r)
	if !ok {
		web.JSONErr(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	entity, err := store.CreateEntity(tx, account.ID)
	if err != nil {
		log.Printf("cannot create entity: %s", err)
		web.JSONErr(w, "Cannot create entity", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		web.JSONErr(w, "Cannot create entity", http.StatusServiceUnavailable)
		return
	}
	web.JSONRedirect(w, "/e/"+entity.Key, http.StatusFound)
}

func handleRenderBanner(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	entityID := web.Args(ctx).ByIndex(0)
	db := store.DB(ctx)
	entity, err := store.EntityByKey(db, entityID)
	if err != nil {
		if err == store.ErrNotFound {
			text := fmt.Sprintf("Entity %q does not exist", entityID)
			web.JSONErr(w, text, http.StatusNotFound)
		} else {
			log.Printf("cannot get entity vote: %s", err)
			web.StdJSONErr(w, http.StatusInternalServerError)
		}
		return
	}

	tag, err := renderCount(entity.Votes)
	if err != nil {
		log.Printf("cannot render tag: %s", err)
		web.JSONErr(w, "Cannot render tag", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if _, err := io.Copy(w, tag); err != nil {
		log.Printf("cannot write response: %s", err)
	}
}

func handleAddVote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tx, err := store.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot create transcation: %s", err)
		web.StdJSONErr(w, http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	account, ok := Authenticated(tx, r)
	if !ok {
		web.JSONErr(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	entityID := web.Args(ctx).ByIndex(0)
	if _, err := store.AddVote(tx, entityID, account.ID); err != nil {
		if err != store.ErrConflict {
			log.Printf("cannot add vote: %s", err)
		}
	} else {
		if err := tx.Commit(); err != nil {
			log.Printf("cannot commit transaction: %s", err)
			web.StdJSONErr(w, http.StatusServiceUnavailable)
			return
		}
	}

	log.Printf("referer: %q", r.Referer())
	log.Printf("header: %#v", r.Header)
	if ref := r.Referer(); ref != "" {
		web.JSONRedirect(w, ref, http.StatusFound)
	} else {
		web.JSONRedirect(w, "/", http.StatusFound)
	}
}

func handleDelVote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tx, err := store.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot create transcation: %s", err)
		web.StdJSONErr(w, http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	account, ok := Authenticated(tx, r)
	if !ok {
		web.JSONErr(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	entityID := web.Args(ctx).ByIndex(0)
	if err := store.DelVote(tx, entityID, account.ID); err != nil {
		if err == store.ErrNotFound {
			web.JSONErr(w, "Not voted on", http.StatusNotFound)
		} else {
			log.Printf("cannot delete vote: %s", err)
			web.JSONErr(w, "Cannot delete vote", http.StatusInternalServerError)
		}
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		web.StdJSONErr(w, http.StatusServiceUnavailable)
		return
	}

	if ref := r.Referer(); ref != "" {
		web.JSONRedirect(w, ref, http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
