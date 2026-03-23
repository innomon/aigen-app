package relationdbdao

import (
	"fmt"

	"github.com/innomon/aigen-cms/core/descriptors"
)

func CreateDao(provider descriptors.DatabaseProvider, connectionString string) (IPrimaryDao, error) {
	switch provider {
	case descriptors.Postgres:
		return NewPostgresDao(connectionString)
	case descriptors.SQLite:
		return NewSqliteDao(connectionString)
	case descriptors.MySQL:
		return nil, fmt.Errorf("MySQL provider not implemented yet")
	case descriptors.SQLServer:
		return nil, fmt.Errorf("SQLServer provider not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", provider)
	}
}
