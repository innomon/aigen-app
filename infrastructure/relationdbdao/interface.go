package relationdbdao

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-app/utils/datamodels"
)

const RecordsTable = "aigen_records"

type IPrimaryDao interface {
	Save(ctx context.Context, rec datamodels.RecJSON) error
	Get(ctx context.Context, namespace, key string) (*datamodels.RecJSON, error)
	Delete(ctx context.Context, namespace, key string) error
	List(ctx context.Context, namespace string, filters []datamodels.Filter, pagination datamodels.Pagination, sorts []datamodels.Sort) ([]datamodels.RecJSON, int64, error)
	
	EnsureTable(ctx context.Context) error
	Begin(ctx context.Context) (*sql.Tx, error)
	GetBuilder() squirrel.StatementBuilderType
	GetDb() *sql.DB
	Ping(ctx context.Context) error
	Close() error
}
