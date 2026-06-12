package relationdbdao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-app/utils/datamodels"
	surrealdb "github.com/surrealdb/surrealdb.go"
)

type SurrealDBDao struct {
	client *surrealdb.DB
}

func parseSurrealDBConnString(connStr string) (endpoint, username, password, ns, db string, err error) {
	rawStr := connStr
	if strings.HasPrefix(rawStr, "surreal://") {
		rawStr = "ws://" + strings.TrimPrefix(rawStr, "surreal://")
	} else if strings.HasPrefix(rawStr, "surrealdb://") {
		rawStr = "ws://" + strings.TrimPrefix(rawStr, "surrealdb://")
	}

	u, err := url.Parse(rawStr)
	if err != nil {
		return "", "", "", "", "", err
	}

	endpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 1 && parts[0] != "" {
		ns = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		db = parts[1]
	}

	if ns == "" {
		ns = "aigen"
	}
	if db == "" {
		db = "aigen"
	}
	if username == "" {
		username = "root"
	}
	if password == "" {
		password = "root"
	}

	return endpoint, username, password, ns, db, nil
}

func NewSurrealDBDao(connectionString string) (*SurrealDBDao, error) {
	endpoint, username, password, ns, db, err := parseSurrealDBConnString(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	ctx := context.Background()
	client, err := surrealdb.FromEndpointURLString(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}

	_, err = client.SignIn(ctx, surrealdb.Auth{
		Username: username,
		Password: password,
	})
	if err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to sign in to surrealdb: %w", err)
	}

	// Define namespace if it doesn't exist
	_, _ = surrealdb.Query[any](ctx, client, fmt.Sprintf("DEFINE NAMESPACE `%s`", ns), nil)

	// Use namespace
	if err := client.Use(ctx, ns, ""); err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to use namespace: %w", err)
	}

	// Define database if it doesn't exist
	_, _ = surrealdb.Query[any](ctx, client, fmt.Sprintf("DEFINE DATABASE `%s`", db), nil)

	// Use both namespace and database
	if err := client.Use(ctx, ns, db); err != nil {
		client.Close(ctx)
		return nil, fmt.Errorf("failed to use database: %w", err)
	}

	return &SurrealDBDao{
		client: client,
	}, nil
}

// EnsureTable does nothing for SurrealDB (tables are implicit on write)
func (d *SurrealDBDao) EnsureTable(ctx context.Context) error {
	return nil
}

func (d *SurrealDBDao) Save(ctx context.Context, rec datamodels.RecJSON) error {
	rec.MetaData.Revision++

	if rec.Tmstamp.IsZero() {
		rec.Tmstamp = time.Now()
	}

	idStr := fmt.Sprintf("aigen_records:`%s___%s`", rec.Namespace, rec.Key)
	query := fmt.Sprintf("UPSERT %s CONTENT $content", idStr)
	_, err := surrealdb.Query[any](ctx, d.client, query, map[string]any{"content": rec})
	return err
}

func (d *SurrealDBDao) SaveConditional(ctx context.Context, rec datamodels.RecJSON, expectedRevision int64) error {
	rec.MetaData.Revision = expectedRevision + 1

	if rec.Tmstamp.IsZero() {
		rec.Tmstamp = time.Now()
	}

	idStr := fmt.Sprintf("aigen_records:`%s___%s`", rec.Namespace, rec.Key)
	query := fmt.Sprintf("UPDATE %s CONTENT $content WHERE metadata.revision = $expected", idStr)
	vars := map[string]any{
		"content":  rec,
		"expected": expectedRevision,
	}

	results, err := surrealdb.Query[[]datamodels.RecJSON](ctx, d.client, query, vars)
	if err != nil {
		return err
	}
	if len(*results) == 0 {
		return fmt.Errorf("optimistic concurrency conflict: expected revision %d", expectedRevision)
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" {
		return fmt.Errorf("query status not OK: %s", qRes.Status)
	}
	if len(qRes.Result) == 0 {
		return fmt.Errorf("optimistic concurrency conflict: expected revision %d", expectedRevision)
	}
	return nil
}

func (d *SurrealDBDao) Get(ctx context.Context, namespace, key string) (*datamodels.RecJSON, error) {
	idStr := fmt.Sprintf("aigen_records:`%s___%s`", namespace, key)
	query := fmt.Sprintf("SELECT * FROM %s", idStr)

	results, err := surrealdb.Query[[]datamodels.RecJSON](ctx, d.client, query, nil)
	if err != nil {
		return nil, err
	}
	if len(*results) == 0 {
		return nil, nil
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" {
		return nil, fmt.Errorf("query status not OK: %s", qRes.Status)
	}
	if len(qRes.Result) == 0 {
		return nil, nil
	}
	return &qRes.Result[0], nil
}

func (d *SurrealDBDao) Delete(ctx context.Context, namespace, key string) error {
	idStr := fmt.Sprintf("aigen_records:`%s___%s`", namespace, key)
	query := fmt.Sprintf("DELETE %s", idStr)
	_, err := surrealdb.Query[any](ctx, d.client, query, nil)
	return err
}

func (d *SurrealDBDao) List(ctx context.Context, namespace string, filters []datamodels.Filter, pagination datamodels.Pagination, sorts []datamodels.Sort) ([]datamodels.RecJSON, int64, error) {
	whereParts := []string{"namespace = $namespace"}
	args := map[string]any{
		"namespace": namespace,
	}

	argCounter := 0
	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				argName := fmt.Sprintf("arg_%d", argCounter)
				argCounter++
				if len(c.Values) == 1 {
					whereParts = append(whereParts, fmt.Sprintf("rec.%s = $%s", f.FieldName, argName))
					args[argName] = c.Values[0]
				} else {
					whereParts = append(whereParts, fmt.Sprintf("rec.%s IN $%s", f.FieldName, argName))
					args[argName] = c.Values
				}
			}
		}
	}

	queryParts := []string{
		"SELECT * FROM aigen_records WHERE " + strings.Join(whereParts, " AND "),
	}

	var sortParts []string
	for _, sort := range sorts {
		order := "ASC"
		if sort.Order == datamodels.SortOrderDesc {
			order = "DESC"
		}
		sortParts = append(sortParts, fmt.Sprintf("rec.%s %s", sort.Field, order))
	}
	if len(sortParts) > 0 {
		queryParts = append(queryParts, "ORDER BY "+strings.Join(sortParts, ", "))
	}

	if pagination.Limit != nil {
		if l, err := strconv.ParseUint(*pagination.Limit, 10, 64); err == nil {
			queryParts = append(queryParts, fmt.Sprintf("LIMIT %d", l))
		}
	}
	if pagination.Offset != nil {
		if o, err := strconv.ParseUint(*pagination.Offset, 10, 64); err == nil {
			queryParts = append(queryParts, fmt.Sprintf("START %d", o))
		}
	}

	query := strings.Join(queryParts, " ")
	results, err := surrealdb.Query[[]datamodels.RecJSON](ctx, d.client, query, args)
	if err != nil {
		return nil, 0, err
	}
	if len(*results) == 0 {
		return nil, 0, fmt.Errorf("no query results returned")
	}
	qRes := (*results)[0]
	if qRes.Status != "OK" {
		return nil, 0, fmt.Errorf("query status not OK: %s", qRes.Status)
	}

	// Count query to get total
	countWhereParts := []string{"namespace = $namespace"}
	countArgs := map[string]any{
		"namespace": namespace,
	}

	argCounter = 0
	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				argName := fmt.Sprintf("arg_%d", argCounter)
				argCounter++
				if len(c.Values) == 1 {
					countWhereParts = append(countWhereParts, fmt.Sprintf("rec.%s = $%s", f.FieldName, argName))
					countArgs[argName] = c.Values[0]
				} else {
					countWhereParts = append(countWhereParts, fmt.Sprintf("rec.%s IN $%s", f.FieldName, argName))
					countArgs[argName] = c.Values
				}
			}
		}
	}

	countParts := []string{
		"SELECT count() AS count FROM aigen_records WHERE " + strings.Join(countWhereParts, " AND "),
		"GROUP ALL",
	}

	countQuery := strings.Join(countParts, " ")
	countResults, err := surrealdb.Query[[]struct {
		Count int64 `json:"count"`
	}](ctx, d.client, countQuery, countArgs)
	var total int64
	if err == nil && len(*countResults) > 0 {
		cRes := (*countResults)[0]
		if cRes.Status == "OK" && len(cRes.Result) > 0 {
			total = cRes.Result[0].Count
		}
	}

	return qRes.Result, total, nil
}

func (d *SurrealDBDao) GetDb() *sql.DB {
	return nil
}

func (d *SurrealDBDao) GetBuilder() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder
}

func (d *SurrealDBDao) Begin(ctx context.Context) (*sql.Tx, error) {
	return nil, errors.New("transactions not supported on SurrealDB via IPrimaryDao")
}

func (d *SurrealDBDao) Ping(ctx context.Context) error {
	// Ping by executing a simple query
	_, err := surrealdb.Query[any](ctx, d.client, "INFO FOR DB", nil)
	return err
}

func (d *SurrealDBDao) Close() error {
	return d.client.Close(context.Background())
}
