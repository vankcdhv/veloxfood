package database

import (
	"errors"
	"fmt"

	"project/pkg/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateUp applies all pending golang-migrate migrations from
// migrationsPath. An already-current schema (ErrNoChange) is not an error.
// The postgres driver takes an advisory lock, so multiple replicas starting
// concurrently do not race each other; a dirty version fails fast on purpose.
func MigrateUp(cfg config.DatabaseConfig, migrationsPath string) error {
	if migrationsPath == "" {
		return fmt.Errorf("migrate: migrations_path is required when auto_migrate is enabled")
	}

	ssl := cfg.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, ssl)

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("migrate: init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}
