package webhooks

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/husio/x/auth"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/core"
	"github.com/husio/x/votehub/votes"
	"github.com/husio/x/web"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func HandleListWebhooks(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	opts := github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	repositories, _, err := client.Repositories.ListByUser(account.Login, &opts)
	if err != nil {
		panic(err)
	}

	context := struct {
		Repositories []github.Repository
	}{
		Repositories: repositories,
	}
	core.Render(tmpl, w, "webhook-list", context)
}

func HandleCreateWebhooks(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := pg.DB(ctx)
	account, ok := auth.AuthRequired(db, w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("cannot parse form: %s", err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}
	var repositories []string
	for name := range r.Form {
		if !strings.HasPrefix(name, "repository-") {
			continue
		}
		repositories = append(repositories, name[11:])
	}
	if len(repositories) == 0 {
		log.Printf("no repositories to create: %v", r.Form)
		http.Redirect(w, r, "/webhooks/create", http.StatusFound)
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

	for _, repo := range repositories {
		_, _, err := client.Repositories.CreateHook(account.Login, repo, &github.Hook{
			Name:   github.String("web"),
			Active: github.Bool(true),
			Events: []string{"issues"},
			Config: map[string]interface{}{
				"url":          "https://example.com/webhooks", // TODO
				"secret":       "TODO-secret",                  // TODO
				"content_type": "json",
			},
		})
		if err != nil {
			log.Printf("cannot create %q hook: %s", repo, err) // TODO - duplicates?
		}
	}

	http.Redirect(w, r, "/webhooks/create", http.StatusFound)
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

	counter, err := votes.CreateCounter(tx, votes.Counter{
		Description: fmt.Sprintf("Issue: %s", input.Issue.Title),
		OwnerID:     input.Repository.Owner.ID,
		URL:         input.Issue.URL,
	})
	if err != nil {
		log.Printf("cannot create counter for %q issue: %s", input.Issue.URL, err)
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	// TODO replace tag if present
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