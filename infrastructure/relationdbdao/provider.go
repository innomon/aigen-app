package relationdbdao

import (
	"strings"
)

func CreateDao(connectionString string) (IPrimaryDao, error) {
	if strings.HasPrefix(connectionString, "postgres://") || strings.Contains(connectionString, "user=") {
		return NewPostgresDao(connectionString)
	}
	return NewSqliteDao(connectionString)
}
