package relationdbdao

import (
	"context"
	"database/sql"

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
