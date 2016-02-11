package webhooks

import (
	"fmt"
	"strings"
	"time"

	"github.com/husio/x/storage/pg"
)

type Webhook struct {
	WebhookID          int    `db:"webhook_id"`
	RepositoryID       int    `db:"repository_id"`
	RepositoryFullName string `db:"repository_full_name"`
	Created            time.Time
}

func CreateWebhook(g pg.Getter, w Webhook) (*Webhook, error) {
	if w.Created.IsZero() {
		w.Created = time.Now()
	}
	err := g.Get(&w, `
		INSERT INTO webhooks (webhook_id, repository_id, repository_full_name, created)
		VALUES ($1, $2, $3, $4)
		RETURNING *
	`, w.WebhookID, w.RepositoryID, w.RepositoryFullName, w.Created)
	return &w, pg.CastErr(err)
}

func WebhooksByRepositoryName(s pg.Selector, names []string) ([]*Webhook, error) {
	var args []interface{}
	var chunks []string

	chunks = append(chunks, "(")
	for i, name := range names {
		chunks = append(chunks, fmt.Sprintf("$%d", i+1), ",")
		args = append(args, name)
	}
	chunks = chunks[:len(chunks)-1]
	chunks = append(chunks, ")")

	query := `
		SELECT * FROM webhooks
		WHERE repository_full_name IN ` + strings.Join(chunks, "")

	var hooks []*Webhook
	err := s.Select(&hooks, query, args...)
	return hooks, pg.CastErr(err)
}
