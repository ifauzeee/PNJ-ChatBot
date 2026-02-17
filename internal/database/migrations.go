package database

import (
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

//go:embed all:migrations
var migrationFiles embed.FS

func runMigrations(db *sqlx.DB, dbType string) error {
	var migrationsPath string

	if dbType == "postgres" {
		migrationsPath = "migrations/postgres"
	} else {
		migrationsPath = "migrations/sqlite"
	}

	d, err := iofs.New(migrationFiles, migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	var m *migrate.Migrate
	if dbType == "postgres" {
		driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("failed to create postgres driver: %w", err)
		}
		m, err = migrate.NewWithInstance("iofs", d, "postgres", driver)
		if err != nil {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
	} else {
		driver, err := sqlite.WithInstance(db.DB, &sqlite.Config{})
		if err != nil {
			return fmt.Errorf("failed to create sqlite driver: %w", err)
		}
		m, err = migrate.NewWithInstance("iofs", d, "sqlite", driver)
		if err != nil {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Printf("✅ [%s] Migrations applied successfully", dbType)
	return nil
}
