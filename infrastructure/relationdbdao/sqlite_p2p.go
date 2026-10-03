package relationdbdao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-app/utils/datamodels"
	p2ppkg "sqlite-p2p/pkg/p2p"
)

// SqliteP2PDao implements IPrimaryDao on top of sqlite-p2p and pure-Go SQLite engine.
type SqliteP2PDao struct {
	Dao
	engine *p2ppkg.Engine
}

// parseSqliteP2PConnString parses a sqlite-p2p connection URL.
// Formats:
//   sqlite-p2p:///path/to/db.sqlite?topic=...&wal=true&crypto=false
//   p2p:///path/to/db.sqlite
//   sqlite-p2p://:memory:
//   /path/to/file.db
func parseSqliteP2PConnString(connStr string) (p2ppkg.EngineOptions, error) {
	opts := p2ppkg.EngineOptions{
		EnableWAL: true,
	}

	raw := connStr
	if raw == "" || raw == "sqlite-p2p://" || raw == "p2p://" || raw == "default" {
		opts.DBPath = "data/aigen.db"
		return opts, nil
	}

	if raw == ":memory:" || raw == "sqlite-p2p://:memory:" || raw == "p2p://:memory:" {
		opts.DBPath = ":memory:"
		opts.EnableWAL = false
		return opts, nil
	}

	var parsedQuery url.Values
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return opts, fmt.Errorf("failed to parse sqlite-p2p url: %w", err)
		}
		// In sqlite-p2p:///abs/path, u.Path is /abs/path.
		// In sqlite-p2p://relative/path, u.Host is relative and u.Path is /path.
		dbPath := u.Path
		if u.Host != "" && u.Host != "localhost" && u.Host != "." {
			dbPath = u.Host + u.Path
		}
		if dbPath == "" {
			dbPath = "data/aigen.db"
		}
		opts.DBPath = dbPath
		parsedQuery = u.Query()
	} else if strings.Contains(raw, "?") {
		parts := strings.SplitN(raw, "?", 2)
		opts.DBPath = parts[0]
		var err error
		parsedQuery, err = url.ParseQuery(parts[1])
		if err != nil {
			return opts, fmt.Errorf("failed to parse query string: %w", err)
		}
	} else {
		opts.DBPath = raw
	}

	if parsedQuery != nil {
		if walStr := parsedQuery.Get("wal"); walStr != "" {
			opts.EnableWAL = walStr == "1" || strings.ToLower(walStr) == "true"
		}
		if cryptoStr := parsedQuery.Get("crypto"); cryptoStr != "" {
			opts.EnableCrypto = cryptoStr == "1" || strings.ToLower(cryptoStr) == "true"
		}
		if topicStr := parsedQuery.Get("topic"); topicStr != "" {
			var topic [32]byte
			copy(topic[:], []byte(topicStr))
			opts.SwarmTopic = topic
		}
		if portStr := parsedQuery.Get("port"); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				opts.SwarmPort = p
			}
		}
		if bootstrapStr := parsedQuery.Get("bootstrap"); bootstrapStr != "" {
			opts.Bootstrap = strings.Split(bootstrapStr, ",")
		}
	}

	return opts, nil
}

// NewSqliteP2PDao initializes a new SqliteP2PDao.
func NewSqliteP2PDao(connectionString string) (*SqliteP2PDao, error) {
	opts, err := parseSqliteP2PConnString(connectionString)
	if err != nil {
		return nil, err
	}

	engine, err := p2ppkg.OpenEngine(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite-p2p engine: %w", err)
	}

	dao := &SqliteP2PDao{
		Dao: Dao{
			db:      engine.DB(),
			builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
		},
		engine: engine,
	}

	ctx := context.Background()
	if err := dao.EnsureTable(ctx); err != nil {
		_ = engine.Close()
		return nil, fmt.Errorf("failed to initialize sqlite-p2p schema: %w", err)
	}

	return dao, nil
}

// EnsureTable initializes the aigen_records table.
func (d *SqliteP2PDao) EnsureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			namespace TEXT NOT NULL,
			key TEXT NOT NULL,
			rec TEXT NOT NULL CHECK(json_valid(rec)),
			metadata TEXT CHECK(json_valid(metadata)),
			tmstamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (namespace, key)
		) WITHOUT ROWID;
		CREATE INDEX IF NOT EXISTS idx_%s_namespace ON %s (namespace);
	`, RecordsTable, RecordsTable, RecordsTable)
	_, err := d.db.ExecContext(ctx, query)
	return err
}

// Save saves or updates a record.
func (d *SqliteP2PDao) Save(ctx context.Context, rec datamodels.RecJSON) error {
	rec.MetaData.Revision++

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
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(namespace, key)
		DO UPDATE SET rec = excluded.rec, metadata = excluded.metadata, tmstamp = excluded.tmstamp;
	`, RecordsTable)

	tm := rec.Tmstamp
	if tm.IsZero() {
		tm = time.Now().UTC()
	}

	_, err = d.db.ExecContext(ctx, query, rec.Namespace, rec.Key, string(recJSON), string(metaJSON), tm)
	return err
}

// SaveConditional conditionally saves a record based on expected revision.
func (d *SqliteP2PDao) SaveConditional(ctx context.Context, rec datamodels.RecJSON, expectedRevision int64) error {
	rec.MetaData.Revision = expectedRevision + 1

	recJSON, err := json.Marshal(rec.Rec)
	if err != nil {
		return err
	}
	metaJSON, err := json.Marshal(rec.MetaData)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET rec = ?, metadata = ?, tmstamp = ?
		WHERE namespace = ? AND key = ? AND CAST(json_extract(metadata, '$.revision') AS INTEGER) = ?;
	`, RecordsTable)

	tm := rec.Tmstamp
	if tm.IsZero() {
		tm = time.Now().UTC()
	}

	result, err := d.db.ExecContext(ctx, query, string(recJSON), string(metaJSON), tm, rec.Namespace, rec.Key, expectedRevision)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("optimistic concurrency conflict: expected revision %d", expectedRevision)
	}

	return nil
}

// Get retrieves a record by namespace and key.
func (d *SqliteP2PDao) Get(ctx context.Context, namespace, key string) (*datamodels.RecJSON, error) {
	query := fmt.Sprintf(`SELECT namespace, key, rec, metadata, tmstamp FROM %s WHERE namespace = ? AND key = ?`, RecordsTable)
	var rec datamodels.RecJSON
	var recData, metaData string
	err := d.db.QueryRowContext(ctx, query, namespace, key).Scan(&rec.Namespace, &rec.Key, &recData, &metaData, &rec.Tmstamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(recData), &rec.Rec); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(metaData), &rec.MetaData); err != nil {
		return nil, err
	}

	return &rec, nil
}

// Delete removes a record by namespace and key.
func (d *SqliteP2PDao) Delete(ctx context.Context, namespace, key string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE namespace = ? AND key = ?`, RecordsTable)
	_, err := d.db.ExecContext(ctx, query, namespace, key)
	return err
}

// List queries records in a namespace matching filters, sorting, and pagination.
func (d *SqliteP2PDao) List(ctx context.Context, namespace string, filters []datamodels.Filter, pagination datamodels.Pagination, sorts []datamodels.Sort) ([]datamodels.RecJSON, int64, error) {
	sb := d.builder.Select("namespace", "key", "rec", "metadata", "tmstamp").
		From(RecordsTable).
		Where(squirrel.Eq{"namespace": namespace})

	for _, f := range filters {
		if err := ValidateFieldName(f.FieldName); err != nil {
			return nil, 0, err
		}
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				vals := make([]interface{}, len(c.Values))
				copy(vals, c.Values)
				if len(c.Values) == 1 {
					sb = sb.Where(fmt.Sprintf("json_extract(rec, '$.%s') = ?", f.FieldName), vals[0])
				} else {
					sb = sb.Where(fmt.Sprintf("json_extract(rec, '$.%s') IN (%s)", f.FieldName, strings.Repeat("?,", len(c.Values)-1)+"?"), vals...)
				}
			}
		}
	}

	for _, sort := range sorts {
		if err := ValidateFieldName(sort.Field); err != nil {
			return nil, 0, err
		}
		order := "ASC"
		if sort.Order == datamodels.SortOrderDesc {
			order = "DESC"
		}
		sb = sb.OrderBy(fmt.Sprintf("json_extract(rec, '$.%s') %s", sort.Field, order))
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
		var recData, metaData string
		if err := rows.Scan(&rec.Namespace, &rec.Key, &recData, &metaData, &rec.Tmstamp); err != nil {
			return nil, 0, err
		}
		if recData != "" {
			if err := json.Unmarshal([]byte(recData), &rec.Rec); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal rec: %w", err)
			}
		}
		if metaData != "" {
			if err := json.Unmarshal([]byte(metaData), &rec.MetaData); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		results = append(results, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	countSb := d.builder.Select("COUNT(*)").From(RecordsTable).Where(squirrel.Eq{"namespace": namespace})
	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				vals := make([]interface{}, len(c.Values))
				copy(vals, c.Values)
				if len(c.Values) == 1 {
					countSb = countSb.Where(fmt.Sprintf("json_extract(rec, '$.%s') = ?", f.FieldName), vals[0])
				} else {
					countSb = countSb.Where(fmt.Sprintf("json_extract(rec, '$.%s') IN (%s)", f.FieldName, strings.Repeat("?,", len(c.Values)-1)+"?"), vals...)
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

// Close closes the underlying p2p engine and database.
func (d *SqliteP2PDao) Close() error {
	if d.engine != nil {
		return d.engine.Close()
	}
	return d.Dao.Close()
}
