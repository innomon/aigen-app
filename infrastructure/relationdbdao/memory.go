package relationdbdao

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/Masterminds/squirrel"
	"github.com/innomon/aigen-app/utils/datamodels"
)

type MemoryDao struct {
	mu    sync.RWMutex
	data  map[string]map[string]datamodels.RecJSON
}

func NewMemoryDao() *MemoryDao {
	return &MemoryDao{
		data: make(map[string]map[string]datamodels.RecJSON),
	}
}

func (d *MemoryDao) EnsureTable(ctx context.Context) error {
	return nil
}

func (d *MemoryDao) Save(ctx context.Context, rec datamodels.RecJSON) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Normalize Rec to map[string]interface{} for filtering support if it's not already
	if rec.Rec != nil {
		data, err := json.Marshal(rec.Rec)
		if err != nil {
			return err
		}
		
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var m map[string]interface{}
		if err := decoder.Decode(&m); err != nil {
			// If it's not a map (e.g. a string or number), we can't filter it as a map
			// But we still store it.
		} else {
			rec.Rec = m
		}
	}

	if d.data[rec.Namespace] == nil {
		d.data[rec.Namespace] = make(map[string]datamodels.RecJSON)
	}
	d.data[rec.Namespace][rec.Key] = rec
	return nil
}

func (d *MemoryDao) Get(ctx context.Context, namespace, key string) (*datamodels.RecJSON, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.data[namespace] == nil {
		return nil, nil
	}
	rec, ok := d.data[namespace][key]
	if !ok {
		return nil, nil
	}
	return &rec, nil
}

func (d *MemoryDao) Delete(ctx context.Context, namespace, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.data[namespace] != nil {
		delete(d.data[namespace], key)
	}
	return nil
}

func (d *MemoryDao) List(ctx context.Context, namespace string, filters []datamodels.Filter, pagination datamodels.Pagination, sorts []datamodels.Sort) ([]datamodels.RecJSON, int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nsData := d.data[namespace]
	var results []datamodels.RecJSON
	for _, rec := range nsData {
		// Simple filter: only handle 'equals' for now if needed by tests
		match := true
		for _, f := range filters {
			recMap, ok := rec.Rec.(map[string]interface{})
			if !ok {
				match = false
				break
			}
			val, ok := recMap[f.FieldName]
			if !ok {
				match = false
				break
			}
			for _, c := range f.Constraints {
				if c.Match == "equals" {
					found := false
					for _, v := range c.Values {
						if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", v) {
							found = true
							break
						}
					}
					if !found {
						match = false
						break
					}
				}
			}
			if !match {
				break
			}
		}
		if match {
			results = append(results, rec)
		}
	}

	// Sorting
	for _, s := range sorts {
		sort.Slice(results, func(i, j int) bool {
			valI := results[i].Rec.(map[string]interface{})[s.Field]
			valJ := results[j].Rec.(map[string]interface{})[s.Field]

			res := fmt.Sprintf("%v", valI) < fmt.Sprintf("%v", valJ)
			if s.Order == datamodels.SortOrderDesc {
				return !res
			}
			return res
		})
	}

	total := int64(len(results))
	return results, total, nil
}

func (d *MemoryDao) Begin(ctx context.Context) (*sql.Tx, error) {
	return nil, fmt.Errorf("transactions not supported in MemoryDao")
}

func (d *MemoryDao) GetBuilder() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder
}

func (d *MemoryDao) GetDb() *sql.DB {
	return nil
}

func (d *MemoryDao) Ping(ctx context.Context) error {
	return nil
}

func (d *MemoryDao) Close() error {
	return nil
}
