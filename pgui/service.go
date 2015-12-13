package main

import (
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"github.com/husio/x/web"
	"github.com/jmoiron/sqlx"
)

type PgUI struct {
	db *sqlx.DB
	rt *web.Router
}

func NewPgUI(db *sqlx.DB) *PgUI {
	ui := &PgUI{
		db: db,
	}
	ui.rt = web.NewRouter("", web.Routes{
		{"GET", `^/$`, ui.handleQueryList},
		{"GET", `^/{query_id:\d+}$`, ui.handleQueryDetails},
		{"POST", `^/{query_id:\d+}$`, ui.handleQueryList},
	})
	return ui
}

func (p *PgUI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.rt.ServeHTTP(w, r)
}

func (p *PgUI) handleQueryList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	queries, err := ListStoredQueries(p.db, 0, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%#v", queries)
}

func (p *PgUI) handleQueryDetails(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	args := web.Args(ctx)
	queryID, err := strconv.ParseInt(args.ByName("query_id"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tx, err := p.db.Beginx()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	query, err := GetStoredQuery(tx, queryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	results, err := ListStoredResults(tx, query.QueryID, 0, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "%#v\n\n", query)
	for _, r := range results {
		fmt.Fprintf(w, "%#v\n", r)
	}
}

func (p *PgUI) handleQueryCreate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func (p *PgUI) handleQueryExec(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	args := web.Args(ctx)
	queryID, err := strconv.ParseInt(args.ByName("query_id"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	query, err := GetStoredQuery(p.db, queryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	result, err := ExecQuery(p.db, query, "/tmp")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%#v", result)
}

func (p *PgUI) handleResultList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func (p *PgUI) handleResultDetails(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func (p *PgUI) handleResultDownload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}
