package relationdbdao

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/Masterminds/squirrel"
)

type Dao struct {
	db      *sql.DB
	builder squirrel.StatementBuilderType
}

func (d *Dao) GetDb() *sql.DB {
	return d.db
}

func (d *Dao) GetBuilder() squirrel.StatementBuilderType {
	return d.builder
}

func (d *Dao) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *Dao) Close() error {
	return d.db.Close()
}

func (d *Dao) Begin(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}

var validFieldNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

// ValidateFieldName ensures field names used in JSON extraction or sort clauses contain only safe identifier characters.
func ValidateFieldName(fieldName string) error {
	if !validFieldNameRegex.MatchString(fieldName) {
		return fmt.Errorf("invalid field name: %q", fieldName)
	}
	return nil
}

