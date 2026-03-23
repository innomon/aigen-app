package relationdbdao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/utils/datamodels"
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

func (d *PostgresDao) quote(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func (d *PostgresDao) GetColumnDefinitions(ctx context.Context, table string) ([]datamodels.Column, error) {
	query := `SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1`
	rows, err := d.db.QueryContext(ctx, query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []datamodels.Column
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}
		cols = append(cols, datamodels.Column{
			Name: name,
			Type: d.stringToColType(dataType),
		})
	}
	return cols, nil
}

func (d *PostgresDao) CreateTable(ctx context.Context, table string, cols []datamodels.Column) error {
	var parts []string
	var updatedAtField string

	for _, col := range cols {
		if col.Type == datamodels.UpdatedTime {
			updatedAtField = col.Name
		}
		parts = append(parts, fmt.Sprintf(`%s %s`, d.quote(col.Name), d.colTypeToString(col)))
	}

	sqlStr := fmt.Sprintf(`CREATE TABLE %s (%s);`, d.quote(table), strings.Join(parts, ", "))

	if updatedAtField != "" {
		triggerSql := fmt.Sprintf(`
			CREATE OR REPLACE FUNCTION __update_%s_column()
			RETURNS TRIGGER AS $$
			BEGIN
				NEW.%s = timezone('UTC', now());
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;

			CREATE TRIGGER update_%s_%s
			BEFORE UPDATE ON %s
			FOR EACH ROW
			EXECUTE FUNCTION __update_%s_column();
		`, updatedAtField, d.quote(updatedAtField), table, updatedAtField, d.quote(table), updatedAtField)
		sqlStr += triggerSql
	}

	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *PostgresDao) AddColumns(ctx context.Context, table string, cols []datamodels.Column) error {
	var parts []string
	for _, col := range cols {
		parts = append(parts, fmt.Sprintf(`ADD COLUMN %s %s`, d.quote(col.Name), d.colTypeToString(col)))
	}
	sqlStr := fmt.Sprintf(`ALTER TABLE %s %s;`, d.quote(table), strings.Join(parts, ", "))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *PostgresDao) CreateForeignKey(ctx context.Context, table, col, refTable, refCol string) error {
	sqlStr := fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE CASCADE;`,
		d.quote(table), d.quote(fmt.Sprintf("fk_%s_%s", table, col)), d.quote(col), d.quote(refTable), d.quote(refCol))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *PostgresDao) CreateIndex(ctx context.Context, table string, fields []string, isUniq bool) error {
	unique := ""
	if isUniq {
		unique = "UNIQUE"
	}
	quotedFields := make([]string, len(fields))
	for i, f := range fields {
		quotedFields[i] = d.quote(f)
	}
	sqlStr := fmt.Sprintf(`CREATE %s INDEX %s ON %s (%s);`,
		unique, d.quote(fmt.Sprintf("idx_%s_%s", table, strings.Join(fields, "_"))), d.quote(table), strings.Join(quotedFields, ", "))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *PostgresDao) CastDate(field string) string {
	return fmt.Sprintf(`%s::date`, d.quote(field))
}

func (d *PostgresDao) stringToColType(s string) datamodels.ColumnType {
	switch strings.ToLower(s) {
	case "integer", "bigint":
		return datamodels.Int
	case "boolean":
		return datamodels.Boolean
	case "timestamp with time zone", "timestamp without time zone":
		return datamodels.Datetime
	case "text":
		return datamodels.Text
	case "character varying":
		return datamodels.String
	case "numeric", "decimal", "real", "double precision":
		return datamodels.Float
	default:
		return datamodels.String
	}
}

func (d *PostgresDao) colTypeToString(col datamodels.Column) string {
	switch col.Type {
	case datamodels.Id:
		return "SERIAL PRIMARY KEY"
	case datamodels.StringPrimaryKey:
		return fmt.Sprintf("VARCHAR(%d) PRIMARY KEY", col.Length)
	case datamodels.Int:
		return "BIGINT"
	case datamodels.Boolean:
		return "BOOLEAN"
	case datamodels.Datetime:
		return "TIMESTAMP WITH TIME ZONE"
	case datamodels.CreatedTime:
		return "TIMESTAMP WITH TIME ZONE DEFAULT (timezone('UTC', now()))"
	case datamodels.UpdatedTime:
		return "TIMESTAMP WITH TIME ZONE DEFAULT (timezone('UTC', now()))"
	case datamodels.Text:
		return "TEXT"
	case datamodels.String:
		length := col.Length
		if length == 0 {
			length = 255
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case datamodels.Float:
		return "NUMERIC(18,4)"
	default:
		return "TEXT"
	}
}

func (d *PostgresDao) UpdateOnConflict(ctx context.Context, tableName string, data datamodels.Record, keyFields []string) (bool, error) {
	isKey := make(map[string]bool)
	for _, f := range keyFields {
		isKey[f] = true
	}

	var columns, placeholders, updates []string
	var args []interface{}
	var conflictFields []string
	idx := 1

	for _, k := range keyFields {
		columns = append(columns, d.quote(k))
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		args = append(args, data[k])
		conflictFields = append(conflictFields, d.quote(k))
		idx++
	}

	for k, v := range data {
		if isKey[k] {
			continue
		}
		columns = append(columns, d.quote(k))
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		args = append(args, v)
		updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", d.quote(k), d.quote(k)))
		idx++
	}

	sqlStr := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s RETURNING xmax;`,
		d.quote(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "),
		strings.Join(conflictFields, ", "), strings.Join(updates, ", "))

	var xmax int
	err := d.db.QueryRowContext(ctx, sqlStr, args...).Scan(&xmax)
	if err != nil {
		return false, err
	}

	return xmax != 0, nil
}

func (d *PostgresDao) BatchUpdateOnConflict(ctx context.Context, tableName string, records []datamodels.Record, keyFields []string) error {
	if len(records) == 0 {
		return nil
	}

	isKey := make(map[string]bool)
	for _, f := range keyFields {
		isKey[f] = true
	}

	allFields := make([]string, 0, len(records[0]))
	for k := range records[0] {
		allFields = append(allFields, k)
	}

	var columns, updates []string
	for _, f := range allFields {
		columns = append(columns, d.quote(f))
		if !isKey[f] {
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", d.quote(f), d.quote(f)))
		}
	}

	var placeholders []string
	var args []interface{}
	idx := 1
	for _, rec := range records {
		var rowPlaceholders []string
		for _, f := range allFields {
			rowPlaceholders = append(rowPlaceholders, fmt.Sprintf("$%d", idx))
			args = append(args, rec[f])
			idx++
		}
		placeholders = append(placeholders, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	var conflictFields []string
	for _, k := range keyFields {
		conflictFields = append(conflictFields, d.quote(k))
	}

	sqlStr := fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s ON CONFLICT (%s) DO UPDATE SET %s;`,
		d.quote(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "),
		strings.Join(conflictFields, ", "), strings.Join(updates, ", "))

	_, err := d.db.ExecContext(ctx, sqlStr, args...)
	return err
}

func (d *PostgresDao) Increase(ctx context.Context, tableName string, keyConditions datamodels.Record, valueField string, initVal, delta int64) (int64, error) {
	var columns, placeholders, conflictFields []string
	var args []interface{}
	idx := 1
	for k, v := range keyConditions {
		columns = append(columns, d.quote(k))
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		args = append(args, v)
		conflictFields = append(conflictFields, d.quote(k))
		idx++
	}
	columns = append(columns, d.quote(valueField))
	placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
	args = append(args, initVal+delta)
	deltaIdx := idx + 1
	args = append(args, delta)

	sqlStr := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s = %s.%s + $%d RETURNING %s;`,
		d.quote(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "),
		strings.Join(conflictFields, ", "), d.quote(valueField), d.quote(tableName), d.quote(valueField), deltaIdx, d.quote(valueField))

	var result int64
	err := d.db.QueryRowContext(ctx, sqlStr, args...).Scan(&result)
	return result, err
}

func (d *PostgresDao) FetchValues(ctx context.Context, tableName string, keyConditions datamodels.Record, inField string, inValues []interface{}, valueField string) (map[string]interface{}, error) {
	idField := "0"
	if inField != "" {
		idField = d.quote(inField)
	}
	sb := d.builder.Select(idField, d.quote(valueField)).From(d.quote(tableName))
	for k, v := range keyConditions {
		sb = sb.Where(squirrel.Eq{d.quote(k): v})
	}
	if inField != "" && len(inValues) > 0 {
		sb = sb.Where(squirrel.Eq{d.quote(inField): inValues})
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

func (d *PostgresDao) MaxId(ctx context.Context, tableName string, fieldName string) (int64, error) {
	var max sql.NullInt64
	query, args, err := d.builder.Select(fmt.Sprintf("MAX(%s)", d.quote(fieldName))).From(d.quote(tableName)).ToSql()
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

func (d *PostgresDao) RenameTable(ctx context.Context, oldName, newName string) error {
	sqlStr := fmt.Sprintf(`ALTER TABLE %s RENAME TO %s;`, d.quote(oldName), d.quote(newName))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *PostgresDao) RenameColumn(ctx context.Context, table, oldName, newName string) error {
	sqlStr := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s;`, d.quote(table), d.quote(oldName), d.quote(newName))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *PostgresDao) DropForeignKey(ctx context.Context, table, fkName string) error {
	sqlStr := fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;`, d.quote(table), d.quote(fkName))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}
