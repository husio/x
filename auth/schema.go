package auth

import "strings"

func Schema() []string {
	return strings.Split(schema, "---")
}

const schema = `

CREATE TABLE IF NOT EXISTS accounts (
    account_id  INTEGER PRIMARY KEY,
    login       TEXT NOT NULL UNIQUE,
	provider    TEXT NOT NULL, -- oauth2 provider name
    created     TIMESTAMPTZ NOT NULL
);

---

CREATE TABLE IF NOT EXISTS sessions (
    key             TEXT PRIMARY KEY,
    account         INTEGER REFERENCES accounts(account_id),
    created         TIMESTAMPTZ NOT NULL,
	provider        TEXT NOT NULL,
	scopes          TEXT NOT NULL,
    access_token    TEXT NOT NULL
);

`
