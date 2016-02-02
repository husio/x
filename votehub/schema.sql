-- CREATE LANGUAGE plpgsql ;

BEGIN;

CREATE TABLE IF NOT EXISTS accounts (
    account_id  INTEGER PRIMARY KEY, -- the same as github key
    login       TEXT NOT NULL UNIQUE,
    created     TIMESTAMPTZ NOT NULL
);


CREATE TABLE IF NOT EXISTS counters (
    counter_id  SERIAL PRIMARY KEY,
    owner_id    INTEGER NOT NULL REFERENCES accounts(account_id),
    created     TIMESTAMPTZ NOT NULL,
    url         TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    value       INTEGER DEFAULT 0
);


CREATE TABLE IF NOT EXISTS votes (
    counter_id  INTEGER NOT NULL REFERENCES counters(counter_id),
    account_id  INTEGER NOT NULL REFERENCES accounts(account_id),
    created     TIMESTAMPTZ NOT NULL,

    PRIMARY KEY(counter_id, account_id)
);


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


DROP TRIGGER IF EXISTS update_counter_value ON votes;
CREATE TRIGGER update_counter_value_trg AFTER INSERT OR DELETE ON votes
    FOR EACH ROW EXECUTE PROCEDURE update_counter_value();


CREATE TABLE IF NOT EXISTS sessions (
    key             TEXT PRIMARY KEY,
    account         INTEGER REFERENCES accounts(account_id),
    created         TIMESTAMPTZ NOT NULL,
    access_token    TEXT NOT NULL
);


COMMIT;
