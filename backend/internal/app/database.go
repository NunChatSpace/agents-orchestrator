package app

import (
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

// NewDatabase opens the PostgreSQL connection, runs migrations, and seeds the initial user.
func NewDatabase() (*sqlx.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://oagent:secret@localhost:5432/oagent?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sqlx.DB) error {
	migrationsDir := os.Getenv("MIGRATION_DIR")
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	log.Info().Str("dir", migrationsDir).Msg("running migrations")
	return goose.Up(db.DB, migrationsDir)
}
