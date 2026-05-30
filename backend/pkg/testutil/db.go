// Package testutil provides shared helpers for integration tests. It is only
// imported from _test files, so it is never compiled into a service binary.
//
// SetupTestDB isolates integration tests from the dev database: it provisions a
// dedicated "<name>_test" database, migrates it to the latest schema, and hands
// back a connection. Tests therefore never truncate or mutate dev data.
package testutil

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"

	"project/pkg/config"
	"project/pkg/database"
)

// SetupTestDB ensures a dedicated test database exists, migrates it from
// migrationsPath, and returns a GORM connection to it. The test DB name is
// "<cfg.Database.Name>_test", overridable via the TEST_DB_NAME env var.
//
// migrationsPath should be resolved relative to the caller via runtime.Caller
// so the test runs regardless of the working directory.
func SetupTestDB(t *testing.T, cfg *config.Config, migrationsPath string) *gorm.DB {
	t.Helper()

	testName := os.Getenv("TEST_DB_NAME")
	if testName == "" {
		testName = cfg.Database.Name + "_test"
	}

	ensureDatabase(t, cfg.Database, testName)

	// Migrations are the schema source of truth (seed rows + extensions included).
	testDBCfg := cfg.Database
	testDBCfg.Name = testName
	runMigrations(t, testDBCfg, migrationsPath)

	db, err := database.NewPostgresDB(testDBCfg)
	if err != nil {
		t.Fatalf("connect test db %q: %v", testName, err)
	}
	return db
}

// Truncate clears the given tables on a test DB. Safe between tests because it
// only ever runs against the isolated "<name>_test" database.
func Truncate(t *testing.T, db *gorm.DB, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tbl + " CASCADE").Error; err != nil {
			t.Logf("truncate %s: %v", tbl, err)
		}
	}
}

// ensureDatabase creates the test database if it does not yet exist, using the
// configured dev DB as a maintenance connection (its data is never touched —
// only pg_database is queried and CREATE DATABASE is issued).
func ensureDatabase(t *testing.T, devCfg config.DatabaseConfig, testName string) {
	t.Helper()

	admin, err := database.NewPostgresDB(devCfg)
	if err != nil {
		t.Skipf("postgres unavailable (%v) — set SKIP_INTEGRATION=1 to suppress", err)
	}
	defer func() {
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	var exists bool
	if err := admin.Raw(
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", testName,
	).Scan(&exists).Error; err != nil {
		t.Fatalf("check test db existence: %v", err)
	}
	if exists {
		return
	}
	// CREATE DATABASE cannot run inside a transaction; GORM Exec issues it standalone.
	if err := admin.Exec(fmt.Sprintf("CREATE DATABASE %q", testName)).Error; err != nil {
		t.Fatalf("create test db %q (fallback: createdb -h %s -p %d -U %s %s): %v",
			testName, devCfg.Host, devCfg.Port, devCfg.User, testName, err)
	}
}

// runMigrations applies all up migrations from migrationsPath to dbCfg's database.
func runMigrations(t *testing.T, dbCfg config.DatabaseConfig, migrationsPath string) {
	t.Helper()

	ssl := dbCfg.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name, ssl)

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		t.Fatalf("create migrator (%s): %v", migrationsPath, err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate up: %v", err)
	}
}
