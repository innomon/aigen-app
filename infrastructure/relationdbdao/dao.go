package relationdbdao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/utils/datamodels"
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

func (d *Dao) MaxId(ctx context.Context, tableName string, fieldName string) (int64, error) {
	var max sql.NullInt64
	query, args, err := d.builder.Select(fmt.Sprintf("MAX(%s)", fieldName)).From(tableName).ToSql()
	if err != nil {
		return 0, err
	}
	err = d.db.QueryRowContext(ctx, query, args...).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max.Valid {
		return max.Int64, nil
	}
	return 0, nil
}

func (d *Dao) FetchValues(ctx context.Context, tableName string, keyConditions datamodels.Record, inField string, inValues []interface{}, valueField string) (map[string]interface{}, error) {
	idField := "0"
	if inField != "" {
		idField = inField
	}
	sb := d.builder.Select(idField, valueField).From(tableName)
	for k, v := range keyConditions {
		sb = sb.Where(squirrel.Eq{k: v})
	}
	if inField != "" && len(inValues) > 0 {
		sb = sb.Where(squirrel.Eq{inField: inValues})
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]interface{})
	for rows.Next() {
		var key interface{}
		var val interface{}
		if err := rows.Scan(&key, &val); err != nil {
			return nil, err
		}
		result[fmt.Sprintf("%v", key)] = val
	}
	return result, nil
}
