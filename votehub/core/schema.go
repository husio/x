package core

import (
	"fmt"
	"strings"

	"github.com/husio/x/storage/pg"
)

func LoadSchema(e pg.Execer) error {
	for _, query := range strings.Split(schema, "---") {
		if _, err := e.Exec(query); err != nil {
			return &SchemaError{Query: query, Err: err}
		}
	}
	return nil
}

type SchemaError struct {
	Query string
	Err   error
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error: %s", e.Err.Error())
}

const schema = `

CREATE TABLE IF NOT EXISTS accounts (
    account_id  INTEGER PRIMARY KEY, -- the same as github key
    login       TEXT NOT NULL UNIQUE,
    created     TIMESTAMPTZ NOT NULL
);

---

CREATE TABLE IF NOT EXISTS counters (
    counter_id  SERIAL PRIMARY KEY,
    owner_id    INTEGER NOT NULL REFERENCES accounts(account_id),
    created     TIMESTAMPTZ NOT NULL,
    url         TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    value       INTEGER DEFAULT 0
);

---

CREATE TABLE IF NOT EXISTS votes (
    counter_id  INTEGER NOT NULL REFERENCES counters(counter_id),
    account_id  INTEGER NOT NULL REFERENCES accounts(account_id),
    created     TIMESTAMPTZ NOT NULL,

    PRIMARY KEY(counter_id, account_id)
);

---

CREATE OR REPLACE FUNCTION update_counter_value()
RETURNS TRIGGER AS
$$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE counters
            SET value = (SELECT COUNT(*) FROM votes WHERE counter_id = NEW.counter_id)
            WHERE counter_id = NEW.counter_id;
        RETURN NEW;
    END IF;

    UPDATE counters
        SET value = (SELECT COUNT(*) FROM votes WHERE counter_id = OLD.counter_id)
        WHERE counter_id = OLD.counter_id;
    RETURN OLD;

END
$$
LANGUAGE plpgsql;

---

DROP TRIGGER IF EXISTS update_counter_value_trg ON votes;
CREATE TRIGGER update_counter_value_trg AFTER INSERT OR DELETE ON votes
    FOR EACH ROW EXECUTE PROCEDURE update_counter_value();

---

CREATE TABLE IF NOT EXISTS webhooks (
	webhook_id 		INTEGER PRIMARY KEY, -- the same as github id
	repo_full_name  TEXT NOT NULL,
	created         TIMESTAMPTZ NOT NULL
);

---

CREATE TABLE IF NOT EXISTS sessions (
    key             TEXT PRIMARY KEY,
    account         INTEGER REFERENCES accounts(account_id),
    created         TIMESTAMPTZ NOT NULL,
    access_token    TEXT NOT NULL
);

`
