package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	d := &DB{db}

	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	d.Exec("ALTER TABLE chat_queue ADD COLUMN preferred_gender TEXT DEFAULT ''")
	d.Exec("ALTER TABLE users ADD COLUMN year INTEGER DEFAULT 0")
	d.Exec("ALTER TABLE chat_queue ADD COLUMN preferred_year INTEGER DEFAULT 0")
	d.Exec("ALTER TABLE users ADD COLUMN karma INTEGER DEFAULT 0")

	log.Println("✅ Database connected and migrated successfully")
	return d, nil
}

func (d *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE NOT NULL,
			email TEXT DEFAULT '',
			gender TEXT DEFAULT '',
			department TEXT DEFAULT '',
			year INTEGER DEFAULT 0,
			display_name TEXT DEFAULT '',
			karma INTEGER DEFAULT 0,
			is_verified BOOLEAN DEFAULT FALSE,
			is_banned BOOLEAN DEFAULT FALSE,
			report_count INTEGER DEFAULT 0,
			total_chats INTEGER DEFAULT 0,
			state TEXT DEFAULT '',
			state_data TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS verification_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER NOT NULL,
			email TEXT NOT NULL,
			code TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user1_id INTEGER NOT NULL,
			user2_id INTEGER NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME,
			FOREIGN KEY (user1_id) REFERENCES users(telegram_id),
			FOREIGN KEY (user2_id) REFERENCES users(telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS chat_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE NOT NULL,
			preferred_dept TEXT DEFAULT '',
			preferred_gender TEXT DEFAULT '',
			preferred_year INTEGER DEFAULT 0,
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS confessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			author_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			department TEXT DEFAULT '',
			like_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS confession_reactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			confession_id INTEGER NOT NULL,
			telegram_id INTEGER NOT NULL,
			reaction TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(confession_id, telegram_id),
			FOREIGN KEY (confession_id) REFERENCES confessions(id),
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reporter_id INTEGER NOT NULL,
			reported_id INTEGER NOT NULL,
			reason TEXT DEFAULT '',
			chat_session_id INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (reporter_id) REFERENCES users(telegram_id),
			FOREIGN KEY (reported_id) REFERENCES users(telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS blocked_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			blocked_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, blocked_id),
			FOREIGN KEY (user_id) REFERENCES users(telegram_id),
			FOREIGN KEY (blocked_id) REFERENCES users(telegram_id)
		)`,
		`CREATE TABLE IF NOT EXISTS whispers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_id INTEGER NOT NULL,
			target_dept TEXT NOT NULL,
			content TEXT NOT NULL,
			sender_dept TEXT DEFAULT '',
			sender_gender TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (sender_id) REFERENCES users(telegram_id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_department ON users(department)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_active ON chat_sessions(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_users ON chat_sessions(user1_id, user2_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_queue_telegram_id ON chat_queue(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_confessions_created ON confessions(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_codes_telegram ON verification_codes(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports(reported_id)`,
	}

	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %s\nerror: %w", m, err)
		}
	}

	return nil
}
