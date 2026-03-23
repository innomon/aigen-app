package relationdbdao

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type IReplicaDao interface {
	GetColumnDefinitions(ctx context.Context, table string) ([]datamodels.Column, error)
	FetchValues(ctx context.Context, tableName string, keyConditions datamodels.Record, inField string, inValues []interface{}, valueField string) (map[string]interface{}, error)
	MaxId(ctx context.Context, tableName string, fieldName string) (int64, error)
	CastDate(field string) string
	GetBuilder() squirrel.StatementBuilderType
	GetDb() *sql.DB
	Ping(ctx context.Context) error
	Close() error
}

type IPrimaryDao interface {
	IReplicaDao
	Begin(ctx context.Context) (*sql.Tx, error)
	CreateTable(ctx context.Context, table string, cols []datamodels.Column) error
	AddColumns(ctx context.Context, table string, cols []datamodels.Column) error
	CreateForeignKey(ctx context.Context, table, col, refTable, refCol string) error
	CreateIndex(ctx context.Context, table string, fields []string, isUniq bool) error
	UpdateOnConflict(ctx context.Context, tableName string, data datamodels.Record, keyFields []string) (bool, error)
	BatchUpdateOnConflict(ctx context.Context, tableName string, records []datamodels.Record, keyFields []string) error
	Increase(ctx context.Context, tableName string, keyConditions datamodels.Record, valueField string, initVal, delta int64) (int64, error)
	RenameTable(ctx context.Context, oldName, newName string) error
	RenameColumn(ctx context.Context, table, oldName, newName string) error
	DropForeignKey(ctx context.Context, table, fkName string) error
}
