-- CREATE LANGUAGE plpgsql ;

BEGIN;

CREATE TABLE IF NOT EXISTS accounts (
	id INTEGER PRIMARY KEY, -- same as github key
	login TEXT NOT NULL UNIQUE,
	created TIMESTAMPTZ NOT NULL
);


CREATE TABLE IF NOT EXISTS entities (
	key TEXT NOT NULL CHECK (char_length(key) > 8 AND char_length(key) < 64) PRIMARY KEY,
    owner INTEGER NOT NULL REFERENCES accounts(id),
	created TIMESTAMPTZ NOT NULL,
	votes INTEGER DEFAULT 0
);


CREATE TABLE IF NOT EXISTS votes (
	entity TEXT NOT NULL REFERENCES entities(key),
	account INTEGER NOT NULL REFERENCES accounts(id),
	created TIMESTAMPTZ NOT NULL,

	PRIMARY KEY(entity, account)
);


CREATE OR REPLACE FUNCTION update_entity_votes()
RETURNS TRIGGER AS
$$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE entities
            SET votes = (SELECT COUNT(*) FROM votes WHERE entity = NEW.entity)
            WHERE key = NEW.entity;
        RETURN NEW;
    END IF;

    UPDATE entities
        SET votes = (SELECT COUNT(*) FROM votes WHERE entity = OLD.entity)
        WHERE key = OLD.entity;
    RETURN OLD;

END
$$
LANGUAGE plpgsql;


DROP TRIGGER IF EXISTS update_entity_votes_trg ON votes;
CREATE TRIGGER update_entity_votes_trg AFTER INSERT OR DELETE ON votes
    FOR EACH ROW EXECUTE PROCEDURE update_entity_votes();


-- too poor to use memcache
CREATE TABLE IF NOT EXISTS sessions (
    key TEXT PRIMARY KEY,
    account INTEGER REFERENCES accounts(id),
    created TIMESTAMPTZ NOT NULL
);


COMMIT;
