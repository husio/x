package pg

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/husio/x/storage/pg"
)

var SchemaSep = "---"

func LoadSchema(e pg.Execer, path string) error {
	queries, err := readSchema(path)
	if err != nil {
		return fmt.Errorf("cannot read schema: %s", err)
	}
	for _, query := range queries {
		if _, err := e.Exec(query); err != nil {
			return &SchemaError{Query: query, Err: err}
		}
	}
	return nil
}

func readSchema(path string) ([]string, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %s", err)
	}
	b, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %s", err)
	}
	return strings.Split(string(b), SchemaSep)
}

type SchemaError struct {
	Query string
	Err   error
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error: %s", e.Err.Error())
}
