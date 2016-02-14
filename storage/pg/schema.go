package pg

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var SchemaSep = "---"

func MustLoadSchema(e Execer, path string) {
	err := LoadSchema(e, path)
	if err == nil {
		return
	}

	if err, ok := err.(*SchemaError); ok {
		fmt.Fprintf(os.Stderr, "cannot load schema: %s\n%s\n", err.Err, err.Query)
	} else {
		fmt.Fprintf(os.Stderr, "cannot load schema: %s\n", err)
	}
	os.Exit(1)
}

func LoadSchema(e Execer, path string) error {
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
	return strings.Split(string(b), SchemaSep), nil
}

type SchemaError struct {
	Query string
	Err   error
}

func (e *SchemaError) Error() string {
	return fmt.Sprintf("schema error: %s", e.Err.Error())
}
