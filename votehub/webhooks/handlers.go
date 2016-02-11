package webhooks

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/husio/x/auth"
	"github.com/husio/x/storage/pg"
	"github.com/husio/x/votehub/core"
	"github.com/husio/x/votehub/ghub"
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
	if account.Scopes == "" {
		next := web.Reverse(ctx, "login-repo-owner")
		url := fmt.Sprintf("%s?next=%s", next, r.URL.Path)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
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

	client := ghub.Client(ctx, oauth2.NewClient(oauth2.NoContext, ts))

	opts := github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	repositories, err := client.ListRepositories(account.Login, &opts)
	if err != nil {
		log.Printf("cannot list repositories: %s", err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}

	var names []string
	for _, r := range repositories {
		names = append(names, *r.FullName)
	}
	hooks, err := WebhooksByRepositoryName(db, names)
	if err != nil {
		log.Printf("cannot list webhooks %v: %s", names, err)
		stdHTMLResp(w, http.StatusInternalServerError)
		return
	}

	hidx := make(map[string]*Webhook)
	for _, h := range hooks {
		hidx[h.RepositoryFullName] = h
	}

	type RepoWithHook struct {
		*github.Repository
		Hook *Webhook
	}
	var withhooks []*RepoWithHook
	for i := range repositories {
		r := repositories[i]
		withhooks = append(withhooks, &RepoWithHook{
			Repository: &r,
			Hook:       hidx[*r.FullName],
		})
	}

	context := struct {
		Repositories []*RepoWithHook
	}{
		Repositories: withhooks,
	}
	core.Render(w, "webhook_list.html", context)
}

func HandleCreateWebhooks(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := pg.DB(ctx)
	account, ok := auth.AuthRequired(db, w, r)
	if !ok {
		return
	}
	if account.Scopes == "" {
		next := web.Reverse(ctx, "login-repo-owner")
		url := fmt.Sprintf("%s?next=%s", next, r.URL.Path)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
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
		http.Redirect(w, r, web.Reverse(ctx, "webhooks-create"), http.StatusFound)
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

	client := ghub.Client(ctx, oauth2.NewClient(oauth2.NoContext, ts))

	for _, repo := range repositories {
		_, err := client.CreateHook(account.Login, repo, &github.Hook{
			Name:   github.String("web"),
			Active: github.Bool(true),
			Events: []string{"issues"},
			Config: map[string]interface{}{
				"url":          "https://votehub.eu" + web.Reverse(ctx, "webhooks-issues-callback"),
				"secret":       "TODO-secret", // TODO
				"content_type": "json",
			},
		})
		if err != nil {
			log.Printf("cannot create %q hook: %s", repo, err) // TODO - duplicates?
		}
	}

	http.Redirect(w, r, "/webhooks/create", http.StatusFound)
}

func HandleIssuesWebhookCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// TODO: check X-Hub-Signature

	switch ev := r.Header.Get("X-Github-Event"); ev {
	case "ping":
		handleIssuesWebhookCallbackPing(ctx, w, r)
	case "issues":
		handleIssuesWebhookCallback(ctx, w, r)
	default:
		log.Printf("issues handler got %q event", ev)
		web.StdJSONErr(w, http.StatusBadRequest)
	}
}

func handleIssuesWebhookCallbackPing(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var input struct {
		HookID     int       `json:"hook_id"`
		Created    time.Time `json:"created_at"`
		Repository struct {
			ID       int    `json:"id"`
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		web.JSONErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	hook := Webhook{
		WebhookID:          input.HookID,
		Created:            input.Created,
		RepositoryID:       input.Repository.ID,
		RepositoryFullName: input.Repository.FullName,
	}

	if hook, err := CreateWebhook(pg.DB(ctx), hook); err != nil {
		web.JSONErr(w, err.Error(), http.StatusInternalServerError)
	} else {
		web.JSONResp(w, hook, http.StatusOK)
	}
}

// findRepoFullName return repo full name (owner/name) for given url string.
// Example:
//
// https://api.github.com/repos/octocat/Hello-World/hooks/1 => octocat/Hello-World
func findRepoFullName(url string) string {
	match := repoFullNameRx.FindStringSubmatch(url)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

var repoFullNameRx = regexp.MustCompile(`github.com/repos/(.*?)/hooks`)

func handleIssuesWebhookCallback(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var input struct {
		Action string `json:"action"`
		Issue  struct {
			HtmlUrl string `json:"html_url"`
			Number  int    `json:"number"`
			Title   string `json:"title"`
			Body    string `json:"body"`
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
	client := ghub.Client(ctx, oauth2.NewClient(oauth2.NoContext, ts))

	var (
		replaced int
		cerr     error
	)
	body := findTagsRx.ReplaceAllStringFunc(input.Issue.Body, func(tag string) string {
		// do not allow to create more than 20 tags at once
		if cerr != nil || replaced > 20 {
			return ""
		}
		title := strings.TrimSpace(input.Issue.Title)
		if len(title) > 200 {
			title = title[:200]
		}
		desc := fmt.Sprintf("Issue %d: %s", input.Issue.Number, title)
		if len(tag) > 12 && tag[10:len(tag)-2] != "" { // {{votehub (DESC)}}
			extra := strings.TrimSpace(tag[10 : len(tag)-2])
			desc = fmt.Sprintf("%s (%s)", desc, extra)
		}

		counter, err := votes.CreateCounter(tx, votes.Counter{
			Description: desc,
			OwnerID:     input.Repository.Owner.ID,
			URL:         input.Issue.HtmlUrl,
		})
		if err != nil {
			cerr = err
			log.Printf("cannot create counter for %q issue: %s", input.Issue.HtmlUrl, err)
			return ""
		}

		replaced++
		return fmt.Sprintf(" [![votehub](https://votehub.eu%s)](https://votehub.eu%s) ",
			web.Reverse(ctx, "counters-banner-svg", counter.CounterID),
			web.Reverse(ctx, "counters-upvote", counter.CounterID))
	})

	if cerr != nil {
		web.StdJSONErr(w, http.StatusInternalServerError)
		return
	}

	_, err = client.EditIssue(
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

var findTagsRx = regexp.MustCompile(`\{\{votehub ?[^\}]{0,100}\}\}`)
