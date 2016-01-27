BEGIN;


CREATE TABLE IF NOT EXISTS accounts (
	account_id 	INTEGER PRIMARY KEY, -- same as github key
	login 		TEXT NOT NULL UNIQUE,
	created 	TIMESTAMPTZ NOT NULL
);


CREATE TABLE IF NOT EXISTS notes (
	note_id 	SERIAL NOT NULL PRIMARY KEY,
	owner_id    INTEGER NOT NULL REFERENCES accounts(account_id),
	content     TEXT NOT NULL,
	is_public   BOOLEAN NOT NULL DEFAULT false,
	created_at  TIMESTAMPTZ NOT NULL,
	expire_at   TIMESTAMPTZ
);


-- too poor to use memcache
CREATE TABLE IF NOT EXISTS sessions (
    key     TEXT PRIMARY KEY,
    account INTEGER REFERENCES accounts(account_id),
    created TIMESTAMPTZ NOT NULL
);


COMMIT;
