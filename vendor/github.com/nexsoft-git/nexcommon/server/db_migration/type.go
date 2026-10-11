package dbmigration

import (
	"database/sql"

	migrate "github.com/rubenv/sql-migrate"
)

type DBMigration interface {
	DbMigratePostresql(
		db *sql.DB,
		fileMigratePath string,
		schemaName string,
		direction migrate.MigrationDirection,
	) (
		int,
		error,
	)
}
