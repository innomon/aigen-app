package relationdbdao

import (
	"context"
	"testing"
	"time"

	"github.com/innomon/aigen-app/utils/datamodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSqliteP2PConnString(t *testing.T) {
	tests := []struct {
		connStr   string
		wantPath  string
		wantWAL   bool
		wantCrypto bool
		wantPort  int
	}{
		{
			connStr:  "sqlite-p2p://:memory:",
			wantPath: ":memory:",
			wantWAL:  false,
		},
		{
			connStr:  "p2p://data/custom.db?wal=true&crypto=true&port=4000",
			wantPath: "data/custom.db",
			wantWAL:  true,
			wantCrypto: true,
			wantPort: 4000,
		},
		{
			connStr:  "sqlite-p2p:///tmp/aigen.db?wal=0",
			wantPath: "/tmp/aigen.db",
			wantWAL:  false,
		},
		{
			connStr:  "",
			wantPath: "data/aigen.db",
			wantWAL:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.connStr, func(t *testing.T) {
			opts, err := parseSqliteP2PConnString(tt.connStr)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, opts.DBPath)
			assert.Equal(t, tt.wantWAL, opts.EnableWAL)
			if tt.wantCrypto {
				assert.Equal(t, tt.wantCrypto, opts.EnableCrypto)
			}
			if tt.wantPort > 0 {
				assert.Equal(t, tt.wantPort, opts.SwarmPort)
			}
		})
	}
}

func TestSqliteP2PDao(t *testing.T) {
	dao, err := CreateDao("sqlite-p2p://:memory:")
	require.NoError(t, err)
	defer dao.Close()

	ctx := context.Background()

	err = dao.EnsureTable(ctx)
	assert.NoError(t, err)

	rec := datamodels.RecJSON{
		Namespace: "users",
		Key:       "user1",
		Rec: map[string]interface{}{
			"name": "Alice",
			"age":  float64(30),
		},
		MetaData: datamodels.MetaData{
			Revision: 0,
		},
		Tmstamp: time.Now().UTC(),
	}

	// 1. Save
	err = dao.Save(ctx, rec)
	assert.NoError(t, err)

	// 2. Get
	fetched, err := dao.Get(ctx, "users", "user1")
	assert.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "users", fetched.Namespace)
	assert.Equal(t, "user1", fetched.Key)

	recMap, ok := fetched.Rec.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Alice", recMap["name"])
	assert.Equal(t, float64(30), recMap["age"])
	assert.Equal(t, int64(1), fetched.MetaData.Revision)

	// 3. SaveConditional (Success)
	fetched.Rec.(map[string]interface{})["age"] = float64(31)
	err = dao.SaveConditional(ctx, *fetched, 1)
	assert.NoError(t, err)

	fetched2, err := dao.Get(ctx, "users", "user1")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), fetched2.MetaData.Revision)
	assert.Equal(t, float64(31), fetched2.Rec.(map[string]interface{})["age"])

	// 4. SaveConditional (Conflict)
	err = dao.SaveConditional(ctx, *fetched2, 1) // Expecting 1, but it is 2
	assert.Error(t, err)

	// 5. List with filters, sorting, pagination
	rec2 := datamodels.RecJSON{
		Namespace: "users",
		Key:       "user2",
		Rec: map[string]interface{}{
			"name": "Bob",
			"age":  float64(25),
		},
		MetaData: datamodels.MetaData{
			Revision: 0,
		},
		Tmstamp: time.Now().UTC(),
	}
	err = dao.Save(ctx, rec2)
	assert.NoError(t, err)

	// List all
	results, total, err := dao.List(ctx, "users", nil, datamodels.Pagination{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)

	// Filter by age = 25
	filters := []datamodels.Filter{
		{
			FieldName: "age",
			Constraints: []datamodels.Constraint{
				{
					Match:  "equals",
					Values: []interface{}{float64(25)},
				},
			},
		},
	}
	results, total, err = dao.List(ctx, "users", filters, datamodels.Pagination{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, results, 1)
	assert.Equal(t, "user2", results[0].Key)

	// Sort by age ASC
	sorts := []datamodels.Sort{
		{
			Field: "age",
			Order: datamodels.SortOrderAsc,
		},
	}
	results, total, err = dao.List(ctx, "users", nil, datamodels.Pagination{}, sorts)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "user2", results[0].Key) // Bob (25)
	assert.Equal(t, "user1", results[1].Key) // Alice (31)

	// Pagination (Limit = 1, Offset = 1)
	limit := "1"
	offset := "1"
	pagination := datamodels.Pagination{
		Limit:  &limit,
		Offset: &offset,
	}
	results, total, err = dao.List(ctx, "users", nil, pagination, sorts)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, results, 1)
	assert.Equal(t, "user1", results[0].Key) // Alice (31)

	// 6. Delete
	err = dao.Delete(ctx, "users", "user1")
	assert.NoError(t, err)
	err = dao.Delete(ctx, "users", "user2")
	assert.NoError(t, err)

	fetchedDeleted, err := dao.Get(ctx, "users", "user1")
	assert.NoError(t, err)
	assert.Nil(t, fetchedDeleted)
}
