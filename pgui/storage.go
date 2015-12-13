package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"
)

type DB interface {
	Select(interface{}, string, ...interface{}) error
	Get(interface{}, string, ...interface{}) error
	Exec(string, ...interface{}) (sql.Result, error)
}

func GetStoredQuery(db DB, qid int64) (*StoredQuery, error) {
	var sq StoredQuery
	err := db.Get(&sq, `
		SELECT * FROM stored_queries WHERE query_id = $1 LIMIT 1
	`, qid)
	return &sq, err
}

func ListStoredQueries(db DB, offset, limit int) ([]*StoredQuery, error) {
	var res []*StoredQuery
	err := db.Select(res, `
		SELECT * FROM stored_queries
		ORDER BY name DESC
		OFFSET $1 LIMIT $2
	`, offset, limit)
	return res, err
}

func ListStoredResults(db DB, queryID int64, offset, limit int) ([]*StoredResult, error) {
	var res []*StoredResult
	err := db.Select(res, `
		SELECT * FROM stored_result
		WHERE query_id = $1
		ORDER BY created DESC
		OFFSET $2 LIMIT $3
	`, queryID, offset, limit)
	return res, err
}

// ExecQuery execute given stored query and creates new result entry if successful.
func ExecQuery(db DB, q *StoredQuery, workDir string) (*StoredResult, error) {
	var res StoredResult
	start := time.Now()
	path := filepath.Join(workDir, fmt.Sprintf("%d.%d.csv", q.QueryID, start))
	query := fmt.Sprintf("COPY (%s) TO '%s' WITH CSV HEADER", q.Query, path)
	_, err := db.Exec(query)
	if err == nil {
		err = db.Get(&res, `
            INSERT INTO stored_result (query_id, row_count, path, work_start, work_end)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING *
        `, q.QueryID, 0, path, start, time.Now())
	}

	return &res, err
}
