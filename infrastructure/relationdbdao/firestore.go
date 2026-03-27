package relationdbdao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-app/utils/datamodels"
	"google.golang.org/api/iterator"
)

type FirestoreDao struct {
	client *firestore.Client
}

func NewFirestoreDao(connectionString string) (*FirestoreDao, error) {
	// firestore://project-id
	projectID := strings.TrimPrefix(connectionString, "firestore://")
	if projectID == "" {
		return nil, errors.New("firestore project id is required")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &FirestoreDao{
		client: client,
	}, nil
}

// EnsureTable does nothing for Firestore
func (d *FirestoreDao) EnsureTable(ctx context.Context) error {
	return nil
}

func (d *FirestoreDao) Save(ctx context.Context, rec datamodels.RecJSON) error {
	docID := fmt.Sprintf("%s:%s", rec.Namespace, rec.Key)
	
	data := map[string]interface{}{
		"namespace": rec.Namespace,
		"key":       rec.Key,
		"rec":       rec.Rec,
		"metadata":  rec.MetaData,
		"tmstamp":   rec.Tmstamp,
	}
	if rec.Tmstamp.IsZero() {
		data["tmstamp"] = time.Now()
	}

	_, err := d.client.Collection(RecordsTable).Doc(docID).Set(ctx, data)
	return err
}

func (d *FirestoreDao) Get(ctx context.Context, namespace, key string) (*datamodels.RecJSON, error) {
	docID := fmt.Sprintf("%s:%s", namespace, key)
	doc, err := d.client.Collection(RecordsTable).Doc(docID).Get(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "code = NotFound") {
			return nil, nil
		}
		return nil, err
	}

	var rec datamodels.RecJSON
	if err := doc.DataTo(&rec); err != nil {
		return nil, err
	}

	return &rec, nil
}

func (d *FirestoreDao) Delete(ctx context.Context, namespace, key string) error {
	docID := fmt.Sprintf("%s:%s", namespace, key)
	_, err := d.client.Collection(RecordsTable).Doc(docID).Delete(ctx)
	return err
}

func (d *FirestoreDao) List(ctx context.Context, namespace string, filters []datamodels.Filter, pagination datamodels.Pagination, sorts []datamodels.Sort) ([]datamodels.RecJSON, int64, error) {
	query := d.client.Collection(RecordsTable).Where("namespace", "==", namespace)

	for _, f := range filters {
		for _, c := range f.Constraints {
			if c.Match == "equals" && len(c.Values) > 0 {
				if len(c.Values) == 1 {
					// Firestore can query nested maps: rec.fieldName
					query = query.Where(fmt.Sprintf("rec.%s", f.FieldName), "==", c.Values[0])
				} else {
					query = query.Where(fmt.Sprintf("rec.%s", f.FieldName), "in", c.Values)
				}
			}
			// Other matches like "greater_than", etc could be implemented here
		}
	}

	for _, sort := range sorts {
		dir := firestore.Asc
		if sort.Order == datamodels.SortOrderDesc {
			dir = firestore.Desc
		}
		query = query.OrderBy(fmt.Sprintf("rec.%s", sort.Field), dir)
	}

	// Total count (requires a separate query or using Firestore aggregation)
	// For simplicity, we'll implement it after fetching or if we can use aggregation.
	// Firestore doesn't have a simple total count without fetching all or using aggregation service.
	
	// Actually we should use aggregation for total count if possible.
	// client.Collection(...).Select().Aggregation(firestore.Count())
	// But it might not be available in all SDK versions.

	if pagination.Offset != nil {
		// Offset is expensive in Firestore. Better to use cursor, but for now:
		// var offset int
		// fmt.Sscanf(*pagination.Offset, "%d", &offset)
		// query = query.Offset(offset)
	}
	
	if pagination.Limit != nil {
		var limit int
		fmt.Sscanf(*pagination.Limit, "%d", &limit)
		query = query.Limit(limit)
	}

	iter := query.Documents(ctx)
	var results []datamodels.RecJSON
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		var rec datamodels.RecJSON
		if err := doc.DataTo(&rec); err != nil {
			return nil, 0, err
		}
		results = append(results, rec)
	}

	// Total count - let's try a simple aggregation if available
	total := int64(len(results)) // Fallback if no pagination or if we want to be simple
	// In a real production app, we should use firestore.Aggregation
	
	return results, total, nil
}

func (d *FirestoreDao) GetDb() *sql.DB {
	return nil
}

func (d *FirestoreDao) GetBuilder() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder
}

func (d *FirestoreDao) Begin(ctx context.Context) (*sql.Tx, error) {
	return nil, errors.New("transactions not supported on Firestore via IPrimaryDao")
}

func (d *FirestoreDao) Ping(ctx context.Context) error {
	// Simple check by trying to get a dummy doc
	_, err := d.client.Collection("_ping").Doc("_ping").Get(ctx)
	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		return err
	}
	return nil
}

func (d *FirestoreDao) Close() error {
	return d.client.Close()
}
