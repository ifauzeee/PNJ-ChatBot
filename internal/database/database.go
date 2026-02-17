package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pnj-anonymous-bot/internal/logger"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sqlx.DB
	DBType  string
	Builder squirrel.StatementBuilderType
}

func New() (*DB, error) {
	dbType := strings.ToLower(strings.TrimSpace(os.Getenv("DB_TYPE")))
	if dbType == "" {
		dbType = "sqlite"
	}

	var db *sqlx.DB
	var err error

	if dbType == "postgres" {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
		db, err = sqlx.Connect("postgres", connStr)
	} else {
		dbPath := os.Getenv("DB_PATH")
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

func (d *DB) PrepareQuery(query string) string {
	if strings.ToLower(d.DBType) == "postgres" {
		if strings.Contains(query, "INSERT OR IGNORE INTO") {
			query = strings.Replace(query, "INSERT OR IGNORE INTO", "INSERT INTO", 1)
			if !strings.Contains(query, "ON CONFLICT") {
				if strings.Contains(query, "cs_queue") {
					query += " ON CONFLICT (user_id) DO NOTHING"
				} else if strings.Contains(query, "blocked_users") {
					query += " ON CONFLICT (user_id, blocked_id) DO NOTHING"
				} else if strings.Contains(query, "room_members") {
					query += " ON CONFLICT (room_id, telegram_id) DO NOTHING"
				} else {
					query += " ON CONFLICT DO NOTHING"
				}
			}
		}
		if strings.Contains(query, "INSERT OR REPLACE INTO") {
			query = strings.Replace(query, "INSERT OR REPLACE INTO", "INSERT INTO", 1)
			if !strings.Contains(query, "ON CONFLICT") {
				if strings.Contains(query, "cs_sessions") {
					query += " ON CONFLICT (user_id) DO UPDATE SET last_activity = EXCLUDED.last_activity"
				} else if strings.Contains(query, "chat_queue") {
					query += " ON CONFLICT (telegram_id) DO UPDATE SET joined_at = EXCLUDED.joined_at"
				} else if strings.Contains(query, "confession_reactions") {
					query += " ON CONFLICT (confession_id, telegram_id) DO NOTHING"
				} else if strings.Contains(query, "users") {
					query += " ON CONFLICT (telegram_id) DO NOTHING"
				}
			}
		}

		n := 1
		for strings.Contains(query, "?") {
			query = strings.Replace(query, "?", fmt.Sprintf("$%d", n), 1)
			n++
		}
	}
	return query
}

func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.DB.Exec(d.PrepareQuery(query), args...)
}

func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.DB.Query(d.PrepareQuery(query), args...)
}

func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.DB.QueryRow(d.PrepareQuery(query), args...)
}

func (d *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return d.DB.Get(dest, d.PrepareQuery(query), args...)
}

func (d *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return d.DB.Select(dest, d.PrepareQuery(query), args...)
}

func (d *DB) InsertGetID(query string, pkColumn string, args ...interface{}) (int64, error) {
	q := d.PrepareQuery(query)
	if d.DBType == "postgres" {
		var id int64
		err := d.DB.QueryRow(q+" RETURNING "+pkColumn, args...).Scan(&id)
		return id, err
	}

	res, err := d.DB.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
