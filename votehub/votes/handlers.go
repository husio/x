package votes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"github.com/husio/x/auth"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/core"
	"github.com/husio/x/web"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func HandleCreateWebhooks(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	stdHTMLResp(w, http.StatusNotImplemented)
	return

	db := pg.DB(ctx)
	account, ok := auth.AuthRequired(db, w, r)
	if !ok {
		return
	}

	token, err := auth.AccessToken(db, account.AccountID)
	if err != nil {
		log.Printf("cannot get access token for %s: %s", account.AccountID, err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	client := github.NewClient(oauth2.NewClient(oauth2.NoContext, ts))

	repositories, _, err := client.Repositories.ListByUser("husio", nil)
	if err != nil {
		panic(err)
	}

	var public []github.Repository
	for _, repo := range repositories {
		if repo.Private != nil && *repo.Private {
			continue
		}

		if *repo.Name != "x" {
			continue
		}
		public = append(public, repo)
	}

	for _, repo := range public {
		hook, _, err := client.Repositories.CreateHook("husio", *repo.Name, &github.Hook{
			Name: github.String("web"),
			Events: []string{
				"issues",
				"commit_comment",
				"gollum", // any time a Wiki page is updated
			},
			Config: map[string]interface{}{
				"url":          "https://example.com/webhooks",
				"content_type": "json",
			},
		})
		if err != nil {
			log.Printf("cannot create %q hook: %s", *repo.Name, err)
			continue
		}
		fmt.Printf("%+v\n", hook)
	}
}

func HandleIssuesWebhookEvent(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if ev := r.Header.Get("X-Github-Event"); ev != "issues" {
		log.Printf("issues handler got %q event", ev)
		web.StdJSONResp(w, http.StatusBadRequest)
		return
	}

	var input struct {
		Action string `json:"action"`
		Issue  struct {
			URL    string `json:"url"`
			Number int    `json:"number"`
			Title  string `json:"title"`
			Body   string `json:"body"`
		} `json:"issue"`
		Repository struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Owner    struct {
				Login string `json:"login"`
				ID    int    `json:"id"`
			} `json:"owner"`
		} `json:"repository"`
	}

	// TODO: check X-Hub-Signature

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("cannot decode webhook body: %s", err)
		web.JSONErr(w, "cannot decode body", http.StatusBadRequest)
		return
	}

	tx, err := pg.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot start transaction: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	token, err := auth.AccessToken(tx, input.Repository.Owner.ID)
	if err != nil {
		log.Printf("cannot get access token for %d: %s", input.Repository.Owner.ID, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	client := github.NewClient(oauth2.NewClient(oauth2.NoContext, ts))

	counter, err := CreateCounter(tx, Counter{
		Description: fmt.Sprintf("Issue: %s", input.Issue.Title),
		OwnerID:     input.Repository.Owner.ID,
		URL:         input.Issue.URL,
	})
	if err != nil {
		log.Printf("cannot create counter for %q issue: %s", input.Issue.URL, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	body := fmt.Sprintf(`![votehub](https://votehub.eu/v/%d/banner.svg)


`, counter.CounterID) + input.Issue.Body
	_, _, err = client.Issues.Edit(
		input.Repository.Owner.Login,
		input.Repository.Name,
		input.Issue.Number,
		&github.IssueRequest{Body: &body})
	if err != nil {
		log.Printf("cannot update %s, %d issue: %s", input.Repository.FullName, input.Issue.Number, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	web.StdJSONResp(w, http.StatusOK)
}

func HandleUpvote(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tx, err := pg.DB(ctx).Beginx()
	if err != nil {
		log.Printf("cannot create transcation: %s", err)
		web.StdJSONErr(w, http.StatusServiceUnavailable)
		return
	}
	defer tx.Rollback()

	account, ok := auth.Authenticated(tx, r)
	if !ok {
		web.JSONErr(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	counterID := stoi(web.Args(ctx).ByIndex(0))
	if _, err := AddVote(tx, counterID, account.AccountID); err != nil {
		if err == pg.ErrConflict {
			web.StdJSONResp(w, http.StatusConflict)
		} else {
			log.Printf("cannot add vote: %s", err)
			web.StdJSONResp(w, http.StatusInternalServerError)
		}
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("cannot commit transaction: %s", err)
		web.StdJSONErr(w, http.StatusServiceUnavailable)
		return
	}

	web.StdJSONResp(w, http.StatusOK)
}

func HandleRenderSVGBanner(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	counterID := stoi(web.Args(ctx).ByIndex(0))
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

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Expires", zeroTime)
	RenderBadge(w, counter.Value)
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
		core.Render(tmpl, w, "click-upvote-login", context)
		return
	}

	if _, err := AddVote(tx, counterID, account.AccountID); err != nil {
		if err != pg.ErrConflict {
			log.Printf("cannot add vote: %s", err)
		}
	} else {
		if err := tx.Commit(); err != nil {
			log.Printf("cannot commit transaction: %s", err)
			stdHTMLResp(w, http.StatusInternalServerError)
			return
		}
	}

	if ref := r.Referer(); ref != "" {
		// TODO render html page with explanation instead
		http.Redirect(w, r, ref, http.StatusFound)
	} else {
		log.Printf("upvote with no referer: %d: %s", counterID, r.Header.Get("user-agent"))
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
