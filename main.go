package main

import (
	dbmigration "github.com/nexsoft-git/nexcommon/server/db_migration"
	"github.com/nexsoft-git/nexlogger/log"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/william22913/sandbox-svc-code/config"
	"github.com/william22913/sandbox-svc-code/router"
	"github.com/william22913/sandbox-svc-code/server_config"
)

func main() {
	cfg := config.AppConfig

	attribute := server_config.NewServerAttribute(cfg)

	if err := attribute.Init(); err != nil {
		log.Fatal().Err(err).Msg("error found when init server attribute")
	}

	router.InitHttpService(
		cfg,
		attribute.MetricMiddleware,
		attribute.Validator.HttpController,
		attribute.HealthChecker,
	)

	// Route registration must precede StartService, which generates the swagger
	// document from what registration accumulated.
	attribute.InitEndpoint()

	if _, err := dbmigration.NewDBMigration().DbMigratePostresql(
		attribute.DBConnection,
		"sql_migrations/main",
		cfg.Postgresql.DefaultSchema,
		migrate.Up,
	); err != nil {
		log.Fatal().Err(err).Msg("failed to run database migration")
	}

	router.StartService(cfg, attribute.Validator.HttpController)
}
