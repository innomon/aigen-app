package relationdbdao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-cms/utils/datamodels"
	_ "modernc.org/sqlite"
)

type SqliteDao struct {
	Dao
}

func NewSqliteDao(connectionString string) (*SqliteDao, error) {
	db, err := sql.Open("sqlite", connectionString)
	if err != nil {
		return nil, err
	}
	return &SqliteDao{
		Dao: Dao{
			db:      db,
			builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
		},
	}, nil
}

func (d *SqliteDao) quote(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func (d *SqliteDao) GetColumnDefinitions(ctx context.Context, table string) ([]datamodels.Column, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", d.quote(table))
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []datamodels.Column
	for rows.Next() {
		var cid int
		var name, dataType string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &dataType, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, datamodels.Column{
			Name: name,
			Type: d.stringToColType(dataType),
		})
	}
	return cols, nil
}

func (d *SqliteDao) CreateTable(ctx context.Context, table string, cols []datamodels.Column) error {
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
			CREATE TRIGGER %s
			BEFORE UPDATE ON %s
			FOR EACH ROW
			BEGIN
				UPDATE %s SET %s = datetime('now') WHERE id = OLD.id;
			END;
		`, d.quote(fmt.Sprintf("update_%s_%s", table, updatedAtField)), d.quote(table), d.quote(table), d.quote(updatedAtField))
		sqlStr += triggerSql
	}

	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *SqliteDao) AddColumns(ctx context.Context, table string, cols []datamodels.Column) error {
	for _, col := range cols {
		sqlStr := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s;`, d.quote(table), d.quote(col.Name), d.colTypeToString(col))
		if _, err := d.db.ExecContext(ctx, sqlStr); err != nil {
			return err
		}
	}
	return nil
}

func (d *SqliteDao) CreateForeignKey(ctx context.Context, table, col, refTable, refCol string) error {
	return nil
}

func (d *SqliteDao) CreateIndex(ctx context.Context, table string, fields []string, isUniq bool) error {
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

func (d *SqliteDao) CastDate(field string) string {
	return fmt.Sprintf(`Date(%s)`, d.quote(field))
}

func (d *SqliteDao) stringToColType(s string) datamodels.ColumnType {
	s = strings.ToLower(s)
	if strings.Contains(s, "int") {
		return datamodels.Int
	}
	if strings.Contains(s, "real") || strings.Contains(s, "float") || strings.Contains(s, "double") || strings.Contains(s, "numeric") || strings.Contains(s, "decimal") {
		return datamodels.Float
	}
	return datamodels.Text
}

func (d *SqliteDao) colTypeToString(col datamodels.Column) string {
	switch col.Type {
	case datamodels.Id:
		return "INTEGER PRIMARY KEY AUTOINCREMENT"
	case datamodels.StringPrimaryKey:
		return "TEXT PRIMARY KEY"
	case datamodels.Int:
		return "INTEGER"
	case datamodels.Boolean:
		return "INTEGER DEFAULT 0"
	case datamodels.Datetime:
		return "INTEGER"
	case datamodels.CreatedTime:
		return "INTEGER DEFAULT (datetime('now'))"
	case datamodels.UpdatedTime:
		return "INTEGER DEFAULT (datetime('now'))"
	case datamodels.Text:
		return "TEXT"
	case datamodels.String:
		return "TEXT"
	case datamodels.Float:
		return "REAL"
	default:
		return "TEXT"
	}
}

func (d *SqliteDao) UpdateOnConflict(ctx context.Context, tableName string, data datamodels.Record, keyFields []string) (bool, error) {
	isKey := make(map[string]bool)
	for _, f := range keyFields {
		isKey[f] = true
	}

	var columns, placeholders, updates []string
	var args []interface{}
	var conflictFields []string

	for _, k := range keyFields {
		columns = append(columns, d.quote(k))
		placeholders = append(placeholders, "?")
		args = append(args, data[k])
		conflictFields = append(conflictFields, d.quote(k))
	}

	for k, v := range data {
		if isKey[k] {
			continue
		}
		columns = append(columns, d.quote(k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
		updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", d.quote(k), d.quote(k)))
	}

	sqlStr := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s;`,
		d.quote(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "),
		strings.Join(conflictFields, ", "), strings.Join(updates, ", "))

	res, err := d.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (d *SqliteDao) BatchUpdateOnConflict(ctx context.Context, tableName string, records []datamodels.Record, keyFields []string) error {
	for _, rec := range records {
		if _, err := d.UpdateOnConflict(ctx, tableName, rec, keyFields); err != nil {
			return err
		}
	}
	return nil
}

func (d *SqliteDao) Increase(ctx context.Context, tableName string, keyConditions datamodels.Record, valueField string, initVal, delta int64) (int64, error) {
	var columns, placeholders, conflictFields []string
	var args []interface{}
	for k, v := range keyConditions {
		columns = append(columns, d.quote(k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
		conflictFields = append(conflictFields, d.quote(k))
	}
	columns = append(columns, d.quote(valueField))
	placeholders = append(placeholders, "?")
	args = append(args, initVal+delta)
	args = append(args, delta)

	sqlStr := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s = COALESCE(%s.%s, 0) + ? RETURNING %s;`,
		d.quote(tableName), strings.Join(columns, ", "), strings.Join(placeholders, ", "),
		strings.Join(conflictFields, ", "), d.quote(valueField), d.quote(tableName), d.quote(valueField), d.quote(valueField))

	var result int64
	err := d.db.QueryRowContext(ctx, sqlStr, args...).Scan(&result)
	return result, err
}

func (d *SqliteDao) FetchValues(ctx context.Context, tableName string, keyConditions datamodels.Record, inField string, inValues []interface{}, valueField string) (map[string]interface{}, error) {
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

func (d *SqliteDao) MaxId(ctx context.Context, tableName string, fieldName string) (int64, error) {
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

func (d *SqliteDao) RenameTable(ctx context.Context, oldName, newName string) error {
	sqlStr := fmt.Sprintf(`ALTER TABLE %s RENAME TO %s;`, d.quote(oldName), d.quote(newName))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *SqliteDao) RenameColumn(ctx context.Context, table, oldName, newName string) error {
	sqlStr := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s;`, d.quote(table), d.quote(oldName), d.quote(newName))
	_, err := d.db.ExecContext(ctx, sqlStr)
	return err
}

func (d *SqliteDao) DropForeignKey(ctx context.Context, table, fkName string) error {
	return nil
}
