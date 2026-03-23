package relationdbdao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"strings"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-app/utils/datamodels"
	_ "github.com/lib/pq"
)

type PostgresDao struct {
	Dao
}

func NewPostgresDao(connectionString string) (*PostgresDao, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	return &PostgresDao{
		Dao: Dao{
			db:      db,
			builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		},
	}, nil
}

func (d *PostgresDao) EnsureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			namespace TEXT NOT NULL,
			key TEXT NOT NULL,
			rec JSONB NOT NULL,
			metadata JSONB,
			tmstamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (namespace, key)
		);
		CREATE INDEX IF NOT EXISTS idx_%s_namespace ON %s (namespace);
	`, RecordsTable, RecordsTable, RecordsTable)
	_, err := d.db.ExecContext(ctx, query)
	return err
}

func (d *PostgresDao) Save(ctx context.Context, rec datamodels.RecJSON) error {
	recJSON, err := json.Marshal(rec.Rec)
	if err != nil {
		return err
	}
	metaJSON, err := json.Marshal(rec.MetaData)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (namespace, key, rec, metadata, tmstamp)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (namespace, key)
		DO UPDATE SET rec = EXCLUDED.rec, metadata = EXCLUDED.metadata, tmstamp = EXCLUDED.tmstamp;
	`, RecordsTable)

	tm := rec.Tmstamp
	if tm.IsZero() {
		tm = time.Now()
	}

	_, err = d.db.ExecContext(ctx, query, rec.Namespace, rec.Key, recJSON, metaJSON, tm)
	return err
}

func (d *PostgresDao) Get(ctx context.Context, namespace, key string) (*datamodels.RecJSON, error) {
	query := fmt.Sprintf(`SELECT namespace, key, rec, metadata, tmstamp FROM %s WHERE namespace = $1 AND key = $2`, RecordsTable)
	var rec datamodels.RecJSON
	var recData, metaData []byte
	err := d.db.QueryRowContext(ctx, query, namespace, key).Scan(&rec.Namespace, &rec.Key, &recData, &metaData, &rec.Tmstamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(recData, &rec.Rec); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metaData, &rec.MetaData); err != nil {
		return nil, err
	}

	return &rec, nil
}

func (d *PostgresDao) Delete(ctx context.Context, namespace, key string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE namespace = $1 AND key = $2`, RecordsTable)
	_, err := d.db.ExecContext(ctx, query, namespace, key)
	return err
}

func (d *PostgresDao) List(ctx context.Context, namespace string, filters []datamodels.Filter, pagination datamodels.Pagination, sorts []datamodels.Sort) ([]datamodels.RecJSON, int64, error) {
	sb := d.builder.Select("namespace", "key", "rec", "metadata", "tmstamp").
		From(RecordsTable).
		Where(squirrel.Eq{"namespace": namespace})

	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				vals := make([]interface{}, len(c.Values))
				copy(vals, c.Values)
				if len(c.Values) == 1 {
					sb = sb.Where(fmt.Sprintf("rec->>'%s' = ?", f.FieldName), vals[0])
				} else {
					sb = sb.Where(fmt.Sprintf("rec->>'%s' IN (%s)", f.FieldName, strings.Repeat("?,", len(c.Values)-1)+"?"), vals...)
				}
			}
		}
	}

	for _, sort := range sorts {
		order := "ASC"
		if sort.Order == datamodels.SortOrderDesc {
			order = "DESC"
		}
		sb = sb.OrderBy(fmt.Sprintf("rec->>'%s' %s", sort.Field, order))
	}

	if pagination.Limit != nil {
		if l, err := strconv.ParseUint(*pagination.Limit, 10, 64); err == nil {
			sb = sb.Limit(l)
		}
	}
	if pagination.Offset != nil {
		if o, err := strconv.ParseUint(*pagination.Offset, 10, 64); err == nil {
			sb = sb.Offset(o)
		}
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []datamodels.RecJSON
	for rows.Next() {
		var rec datamodels.RecJSON
		var recData, metaData []byte
		if err := rows.Scan(&rec.Namespace, &rec.Key, &recData, &metaData, &rec.Tmstamp); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(recData, &rec.Rec)
		json.Unmarshal(metaData, &rec.MetaData)
		results = append(results, rec)
	}

	countSb := d.builder.Select("COUNT(*)").From(RecordsTable).Where(squirrel.Eq{"namespace": namespace})
	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				vals := make([]interface{}, len(c.Values))
				copy(vals, c.Values)
				if len(c.Values) == 1 {
					countSb = countSb.Where(fmt.Sprintf("rec->>'%s' = ?", f.FieldName), vals[0])
				} else {
					countSb = countSb.Where(fmt.Sprintf("rec->>'%s' IN (%s)", f.FieldName, strings.Repeat("?,", len(c.Values)-1)+"?"), vals...)
				}
			}
		}
	}
	countQuery, countArgs, err := countSb.ToSql()
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = d.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}
