package entries

import (
	"log"
	"net/http"

	"github.com/husio/x/storage/pg"

	"golang.org/x/net/context"
)

func HandleListEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	entries, err := ListEntries(pg.DB(ctx), 100, 0)
	if err != nil {
		log.Printf("cannot get entries: %s", err)
		renderStdResp(w, http.StatusInternalServerError)
		return
	}

	context := struct {
		Entries []*Entry
	}{
		Entries: entries,
	}
	render(w, "entry-list", context, http.StatusOK)
}

func HandleListResources(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	renderStdResp(w, http.StatusNotImplemented)
}

func HandleAddResource(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	renderStdResp(w, http.StatusNotImplemented)
}
