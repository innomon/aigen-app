package relationdbdao

import (
	"strings"
)

func CreateDao(connectionString string) (IPrimaryDao, error) {
	if strings.HasPrefix(connectionString, "postgres://") || strings.Contains(connectionString, "user=") {
		return NewPostgresDao(connectionString)
	}
	if strings.HasPrefix(connectionString, "firestore://") {
		return NewFirestoreDao(connectionString)
	}
	// Strip sqlite:// prefix if present
	dsn := strings.TrimPrefix(connectionString, "sqlite://")
	return NewSqliteDao(dsn)
}
