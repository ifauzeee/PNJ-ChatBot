package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pnj-anonymous-bot/internal/config"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sqlx.DB
	DBType  string
	Builder squirrel.StatementBuilderType
}

func New(cfg *config.Config) (*DB, error) {
	dbType := strings.ToLower(strings.TrimSpace(cfg.DBType))
	if dbType == "" {
		dbType = "sqlite"
	}

	var db *sqlx.DB
	var err error

	if dbType == "postgres" {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.DBHost, cfg.DBPort, cfg.DBUser,
			cfg.DBPassword, cfg.DBName)
		db, err = sqlx.Connect("postgres", connStr)
	} else {
		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = "./data/pnj_anonymous.db"
		}
		db, err = sqlx.Connect("sqlite", dbPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if dbType == "postgres" {
		db.SetMaxOpenConns(100)
	} else {
		db.SetMaxOpenConns(25)
	}
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if dbType == "sqlite" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}

	builder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
	if dbType == "postgres" {
		builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	}

	d := &DB{
		DB:      db,
		DBType:  dbType,
		Builder: builder,
	}

	if err := runMigrations(db, dbType); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("✅ Database connected successfully", zap.String("type", dbType))
	return d, nil
}

func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.DB.Exec(query, args...)
}

func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.Query(query, args...)
}

func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRow(query, args...)
}

func (d *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return d.DB.Get(dest, query, args...)
}

func (d *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return d.DB.Select(dest, query, args...)
}

func (d *DB) ExecBuilder(b squirrel.Sqlizer) (sql.Result, error) {
	query, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}
	return d.DB.Exec(query, args...)
}

func (d *DB) GetBuilder(dest interface{}, b squirrel.Sqlizer) error {
	query, args, err := b.ToSql()
	if err != nil {
		return err
	}
	return d.DB.Get(dest, query, args...)
}

func (d *DB) SelectBuilder(dest interface{}, b squirrel.Sqlizer) error {
	query, args, err := b.ToSql()
	if err != nil {
		return err
	}
	return d.DB.Select(dest, query, args...)
}

func (d *DB) InsertIgnore(builder squirrel.InsertBuilder, conflictCol string) (sql.Result, error) {
	if d.DBType == "postgres" {
		builder = builder.Suffix("ON CONFLICT (" + conflictCol + ") DO NOTHING")
	} else {
		query, args, err := builder.ToSql()
		if err != nil {
			return nil, err
		}
		query = strings.Replace(query, "INSERT INTO", "INSERT OR IGNORE INTO", 1)
		return d.DB.Exec(query, args...)
	}
	return d.ExecBuilder(builder)
}

func (d *DB) InsertReplace(builder squirrel.InsertBuilder, conflictCol string, updateCols ...string) (sql.Result, error) {
	if d.DBType == "postgres" {
		var updateClause string
		if len(updateCols) > 0 {
			updateClause = "ON CONFLICT (" + conflictCol + ") DO UPDATE SET "
			for i, col := range updateCols {
				if i > 0 {
					updateClause += ", "
				}
				updateClause += col + " = EXCLUDED." + col
			}
		} else {
			updateClause = "ON CONFLICT (" + conflictCol + ") DO NOTHING"
		}
		builder = builder.Suffix(updateClause)
	} else {
		query, args, err := builder.ToSql()
		if err != nil {
			return nil, err
		}
		query = strings.Replace(query, "INSERT INTO", "INSERT OR REPLACE INTO", 1)
		return d.DB.Exec(query, args...)
	}
	return d.ExecBuilder(builder)
}

func (d *DB) InsertGetID(builder squirrel.InsertBuilder, pkColumn string) (int64, error) {
	if d.DBType == "postgres" {
		query, args, err := builder.Suffix("RETURNING " + pkColumn).ToSql()
		if err != nil {
			return 0, err
		}
		var id int64
		err = d.DB.QueryRow(query, args...).Scan(&id)
		return id, err
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, err
	}
	res, err := d.DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
