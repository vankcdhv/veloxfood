// cmd/migrate/main.go
// Reads the service config via Viper, builds DSN, runs golang-migrate.
// Usage: go run ./cmd/migrate [<service>] up|down|version|force <v>|goto <v>
// <service> defaults to "user". Each service migrates its own database from
// services/<service>/migrations using config/<service>.yaml (user → config.yaml).
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"project/pkg/config"
)

var migrateCommands = map[string]bool{
	"up": true, "down": true, "version": true, "force": true, "goto": true,
}

func main() {
	args := os.Args[1:]

	// Optional leading <service> token (anything that isn't a command).
	service := "user"
	if len(args) > 0 && !migrateCommands[args[0]] {
		service = args[0]
		args = args[1:]
	}
	if len(args) == 0 {
		slog.Error("[migrate] command required: [<service>] up | down | version | force <v> | goto <v>")
		os.Exit(1)
	}

	configPath := "config/config.yaml"
	if service != "user" {
		configPath = fmt.Sprintf("config/%s.yaml", service)
	}
	migrationsPath := fmt.Sprintf("services/%s/migrations", service)

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("[migrate] failed to load config", "service", service, "path", configPath, "error", err)
		os.Exit(1)
	}

	dsn := buildDSN(cfg)
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		slog.Error("[migrate] failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error("[migrate] source close error", "error", srcErr)
		}
		if dbErr != nil {
			slog.Error("[migrate] db close error", "error", dbErr)
		}
	}()

	cmd := args[0]
	switch cmd {
	case "up":
		slog.Info("[migrate] applying up migrations...")
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("[migrate] up failed", "error", err)
			os.Exit(1)
		}
		printVersion(m)
	case "down":
		slog.Info("[migrate] rolling back one migration step...")
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("[migrate] down failed", "error", err)
			os.Exit(1)
		}
		printVersion(m)
	case "version":
		printVersion(m)
	case "force":
		if len(args) < 2 {
			slog.Error("[migrate] force requires a version number")
			os.Exit(1)
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			slog.Error("[migrate] invalid version", "arg", args[1])
			os.Exit(1)
		}
		if err := m.Force(v); err != nil {
			slog.Error("[migrate] force failed", "error", err)
			os.Exit(1)
		}
		slog.Info("[migrate] forced version", "version", v)
	case "goto":
		if len(args) < 2 {
			slog.Error("[migrate] goto requires a version number")
			os.Exit(1)
		}
		v, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			slog.Error("[migrate] invalid version", "arg", args[1])
			os.Exit(1)
		}
		if err := m.Migrate(uint(v)); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("[migrate] goto failed", "error", err)
			os.Exit(1)
		}
		printVersion(m)
	default:
		slog.Error("[migrate] unknown command", "cmd", cmd)
		slog.Info("[migrate] valid commands: up | down | version | force <v> | goto <v>")
		os.Exit(1)
	}

	slog.Info("[migrate] done")
}

func buildDSN(cfg *config.Config) string {
	db := cfg.Database
	ssl := db.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.User, db.Password, db.Host, db.Port, db.Name, ssl)
}

func printVersion(m *migrate.Migrate) {
	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		slog.Error("[migrate] could not read version", "error", err)
		return
	}
	if errors.Is(err, migrate.ErrNilVersion) {
		slog.Info("[migrate] no migrations applied (version: nil)")
		return
	}
	slog.Info("[migrate] current schema version", "version", v, "dirty", dirty)
}
