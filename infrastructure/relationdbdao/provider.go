package relationdbdao

import (
	"fmt"
	"strings"
)

func CreateDao(connectionString string) (IPrimaryDao, error) {
	if strings.HasPrefix(connectionString, "postgres://") || strings.Contains(connectionString, "user=") {
		return NewPostgresDao(connectionString)
	}
	if strings.HasPrefix(connectionString, "firestore://") {
		return NewFirestoreDao(connectionString)
	}
	if strings.HasPrefix(connectionString, "surreal://") || strings.HasPrefix(connectionString, "surrealdb://") {
		return NewSurrealDBDao(connectionString)
	}
	if strings.HasPrefix(connectionString, "memory://") || connectionString == ":memory:" {

		return NewMemoryDao(), nil
	}
	return nil, fmt.Errorf("unsupported database or invalid connection string: %s. SQLite is NOT supported.", connectionString)
}
