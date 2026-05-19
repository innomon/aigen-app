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

	// Increment revision
	rec.MetaData.Revision++

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

func (d *MemoryDao) SaveConditional(ctx context.Context, rec datamodels.RecJSON, expectedRevision int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.data[rec.Namespace] == nil {
		return fmt.Errorf("namespace %s not found", rec.Namespace)
	}
	existing, ok := d.data[rec.Namespace][rec.Key]
	if !ok {
		return fmt.Errorf("record %s not found in namespace %s", rec.Key, rec.Namespace)
	}

	if existing.MetaData.Revision != expectedRevision {
		return fmt.Errorf("optimistic concurrency conflict: expected revision %d, got %d", expectedRevision, existing.MetaData.Revision)
	}

	// Increment revision
	rec.MetaData.Revision = expectedRevision + 1

	// Normalize Rec
	if rec.Rec != nil {
		data, _ := json.Marshal(rec.Rec)
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var m map[string]interface{}
		if err := decoder.Decode(&m); err == nil {
			rec.Rec = m
		}
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
	// Return a copy to avoid side effects
	copy := rec
	return &copy, nil
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
	if len(sorts) > 0 {
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
	} else {
		// Default stable sort by key
		sort.Slice(results, func(i, j int) bool {
			return results[i].Key < results[j].Key
		})
	}

	total := int64(len(results))

	// Pagination
	if pagination.Offset != nil {
		var offset int
		fmt.Sscanf(*pagination.Offset, "%d", &offset)
		if offset > len(results) {
			results = []datamodels.RecJSON{}
		} else {
			results = results[offset:]
		}
	}
	if pagination.Limit != nil {
		var limit int
		fmt.Sscanf(*pagination.Limit, "%d", &limit)
		if limit < len(results) {
			results = results[:limit]
		}
	}

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
