package votes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/husio/x/auth"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/storage/pgtest"
	"github.com/husio/x/votehub/cache"
	"github.com/husio/x/web"
	"golang.org/x/net/context"
)

func TestHandleListCounters(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "xyz")

	now := time.Now()
	db := pgtest.DB{
		Fatalf: t.Fatalf,
		Stack: []pgtest.ResultMock{
			{"Get", auth.Account{AccountID: 321}, nil},
			{"Select", &[]*Counter{{1, 312, now, "", 4, ""}, {2, 321, now, "", 2, ""}}, nil},
			{"Select", &[]*VoteWithCounter{}, nil},
		},
	}

	ctx := context.Background()
	ctx = pgtest.WithDB(ctx, &db)

	HandleListCounters(ctx, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("responded with %d: %s", w.Code, w.Body)
	}
}

func TestHandleRenderCSVBanner(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)

	db := pgtest.DB{
		Fatalf: t.Fatalf,
		Stack: []pgtest.ResultMock{
			{"Get", &Counter{559, 321, time.Now(), "", 74, "https://foo.com"}, nil},
		},
	}

	ctx := context.Background()
	ctx = cache.WithIntCache(ctx)
	ctx = pgtest.WithDB(ctx, &db)
	ctx = web.WithArgs(ctx, []string{"counter-id"}, []string{"559"})

	HandleRenderSVGBanner(ctx, w, r)

	if w.Code != http.StatusOK {
		t.Errorf("responded with %d: %s", w.Code, w.Body)
	}
	if contentType := w.Header().Get("Content-Type"); contentType != "image/svg+xml" {
		t.Errorf("want svg content type, got %q", contentType)
	}

	// executing the same handler again should not use database but int cache
	w = httptest.NewRecorder()
	r, _ = http.NewRequest("GET", "/", nil)
	HandleRenderSVGBanner(ctx, w, r)
}

func TestHandleClickUpvote_Logged(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "xyz")
	const referer = "https://github.com/foobar"
	r.Header.Set("Referer", referer)

	db := pgtest.DB{
		Fatalf: t.Fatalf,
		Stack: []pgtest.ResultMock{
			{"Get", auth.Account{AccountID: 321}, nil},
			{"Get", Vote{559, 123, time.Now()}, nil},
		},
	}

	ctx := context.Background()
	ctx = cache.WithIntCache(ctx)
	ctx = pgtest.WithDB(ctx, &db)
	ctx = web.WithArgs(ctx, []string{"counter-id"}, []string{"559"})

	HandleClickUpvote(ctx, w, r)

	if w.Code != http.StatusFound {
		t.Errorf("responded with %d: %s", w.Code, w.Body)
	}
	if loc := w.Header().Get("Location"); loc != referer {
		t.Errorf("unexpected redirect location: want %q, got %q", referer, loc)
	}
}

func TestHandleClickUpvoteNotExisting_Logged(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "xyz")
	const referer = "https://github.com/foobar"
	r.Header.Set("Referer", referer)

	db := pgtest.DB{
		Fatalf: t.Fatalf,
		Stack: []pgtest.ResultMock{
			{"Get", auth.Account{AccountID: 321}, nil},
			{"Get", nil, pg.ErrNotFound},
		},
	}

	ctx := context.Background()
	ctx = cache.WithIntCache(ctx)
	ctx = pgtest.WithDB(ctx, &db)
	ctx = web.WithArgs(ctx, []string{"counter-id"}, []string{"559"})

	HandleClickUpvote(ctx, w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("responded with %d: %s", w.Code, w.Body)
	}
}

func TestHandleClickUpvote_NotLogged(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "xyz")
	const referer = "https://github.com/foobar"
	r.Header.Set("Referer", referer)

	db := pgtest.DB{
		Fatalf: t.Fatalf,
		Stack: []pgtest.ResultMock{
			{"Get", nil, pg.ErrNotFound},
			{"Get", Counter{CounterID: 559}, nil},
		},
	}

	ctx := context.Background()
	ctx = pgtest.WithDB(ctx, &db)
	ctx = web.WithArgs(ctx, []string{"counter-id"}, []string{"559"})

	HandleClickUpvote(ctx, w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("responded with %d: %s", w.Code, w.Body)
	}
}

func TestHandleClickUpvoteNotExisting_NotLogged(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "xyz")
	const referer = "https://github.com/foobar"
	r.Header.Set("Referer", referer)

	db := pgtest.DB{
		Fatalf: t.Fatalf,
		Stack: []pgtest.ResultMock{
			{"Get", nil, pg.ErrNotFound},
			{"Get", nil, pg.ErrNotFound},
		},
	}

	ctx := context.Background()
	ctx = pgtest.WithDB(ctx, &db)
	ctx = web.WithArgs(ctx, []string{"counter-id"}, []string{"559"})

	HandleClickUpvote(ctx, w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("responded with %d: %s", w.Code, w.Body)
	}
}
