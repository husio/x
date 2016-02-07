package webhooks

import (
	"time"

	"github.com/husio/x/storage/pg"
	"github.com/jmoiron/sqlx"
)

type Webhook struct {
	WebhookID    int    `db:"webhook_id"`
	RepoFullName string `db:"repo_full_name"`
	Created      time.Time
}

func CreateWebhook(g pg.Getter, w Webhook) (*Webhook, error) {
	if w.Created.IsZero() {
		w.Created = time.Now()
	}
	err := g.Get(&w, `
		INSERT INTO webhooks (webhook_id, repo_full_name, created)
		VALUES ($1, $2, $3)
		RETURNING *
	`, w.WebhookID, w.RepoFullName, w.Created)
	return &w, pg.CastErr(err)
}

func FindWebhooks(s pg.Selector, names []string) ([]*Webhook, error) {
	query, args, err := sqlx.In(`
		SELECT * FROM webhooks
		WHERE repo_full_name IN (?)
	`, names)
	if err != nil {
		return nil, err
	}
	var hooks []*Webhook
	err = s.Select(&hooks, query, args)
	return hooks, pg.CastErr(err)
}
