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
	if strings.HasPrefix(connectionString, "memory://") {
		return NewMemoryDao(), nil
	}
	if strings.HasPrefix(connectionString, "sqlite-p2p://") || strings.HasPrefix(connectionString, "p2p://") ||
		strings.HasPrefix(connectionString, "sqlite://") || strings.HasSuffix(connectionString, ".db") ||
		strings.HasSuffix(connectionString, ".sqlite") || connectionString == ":memory:" || connectionString == "" {
		return NewSqliteP2PDao(connectionString)
	}
	return nil, fmt.Errorf("unsupported database or invalid connection string: %s", connectionString)
}

