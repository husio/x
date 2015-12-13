BEGIN;

CREATE TABLE IF NOT EXISTS stored_queries (
    query_id   SERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    created    TIMESTAMPTZ NOT NULL,
    starred    BOOLEAN NOT NULL DEFAULT false,
    query      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stored_result (
    result_id   SERIAL PRIMARY KEY,
    query_id    INTEGER NOT NULL REFERENCES stored_queries(query_id),
    row_count   INTEGER NOT NULL,
    path        TEXT NOT NULL,
    work_start  TIMESTAMPTZ NOT NULL,
    work_end    TIMESTAMPTZ NOT NULL
);

COMMIT;
