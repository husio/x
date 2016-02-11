package votes

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/husio/x/auth"
	"github.com/husio/x/cache"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/core"
	"github.com/husio/x/web"

	"golang.org/x/net/context"
)

func HandleListCounters(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := pg.DB(ctx)
	account, ok := auth.AuthRequired(db, w, r)
	if !ok {
		return
	}

	counters, err := CountersByOwner(db, account.AccountID, 30, 0)
	if err != nil {
		log.Printf("cannot list counter for %d account: %s", account.AccountID, err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}

	votes, err := VotesByOwner(db, account.AccountID, 30, 0)
	if err != nil {
		log.Printf("cannot list counter for %d account: %s", account.AccountID, err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}

	context := struct {
		Counters []*Counter
		Votes    []*VoteWithCounter
	}{
		Counters: counters,
		Votes:    votes,
	}
	core.Render(w, "counters_list.html", context)
}

func HandleRenderSVGBanner(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	counterID := stoi(web.Args(ctx).ByIndex(0))
	key := fmt.Sprintf("counters:%d", counterID)
	cch := cache.Get(ctx)

	var value int64
	if err := cch.Get(key, &value); err != nil {
		counter, err := CounterByID(pg.DB(ctx), counterID)
		if err != nil {
			if err == pg.ErrNotFound {
				stdHTMLResp(w, http.StatusNotFound)
			} else {
				log.Printf("cannot get counter %d vote: %s", counterID, err)
				stdHTMLResp(w, http.StatusInternalServerError)
			}
			return
		}
		value = int64(counter.Value)
		cch.Put(key, value)
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Expires", zeroTime)
	RenderBadge(w, int(value))
}

func stoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

var zeroTime = time.Time{}.Format(http.TimeFormat)

func HandleClickUpvote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tx, err := pg.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot create transcation: %s", err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	counterID := stoi(web.Args(ctx).ByIndex(0))

	account, ok := auth.Authenticated(tx, r)
	if !ok {
		counter, err := CounterByID(tx, counterID)
		if err != nil {
			if err == pg.ErrNotFound {
				stdHTMLResp(w, http.StatusNotFound)
			} else {
				log.Printf("cannot get entity %d: %s", counterID, err)
				stdHTMLResp(w, http.StatusInternalServerError)
			}
			return
		}
		context := struct {
			NextURL string
			Counter *Counter
		}{
			NextURL: r.URL.Path,
			Counter: counter,
		}
		w.WriteHeader(http.StatusUnauthorized)
		core.Render(w, "click_upvote_login.html", context)
		return
	}

	if _, err := AddVote(tx, counterID, account.AccountID); err != nil {
		if err == pg.ErrNotFound {
			stdHTMLResp(w, http.StatusNotFound)
		} else {
			log.Printf("cannot add vote: %s", err)
			stdHTMLResp(w, http.StatusInternalServerError)
		}
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}

	// cache expiration, because new couter happened
	cache.Get(ctx).Del(fmt.Sprintf("counters:%d", counterID))

	if ref := r.Referer(); ref != "" && !strings.HasSuffix(r.URL.Path, "banner.svg") {
		// TODO render html page with explanation instead
		http.Redirect(w, r, ref, http.StatusFound)
	} else {
		log.Printf("upvote with no referer: %d: %s", counterID, r.Header.Get("user-agent"))
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
