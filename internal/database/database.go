package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	DBType string
}

func New() (*DB, error) {
	dbType := strings.ToLower(strings.TrimSpace(os.Getenv("DB_TYPE")))
	if dbType == "" {
		dbType = "sqlite"
	}

	var db *sql.DB
	var err error

	if dbType == "postgres" {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
		db, err = sql.Open("postgres", connStr)
	} else {
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "./data/pnj_anonymous.db"
		}
		db, err = sql.Open("sqlite", dbPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if dbType == "sqlite" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}

	d := &DB{DB: db, DBType: dbType}

	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Printf("✅ [%s] Database connected successfully", dbType)
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
	q := d.PrepareQuery(query)
	return d.DB.Exec(q, args...)
}

func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	q := d.PrepareQuery(query)
	return d.DB.Query(q, args...)
}

func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	q := d.PrepareQuery(query)
	return d.DB.QueryRow(q, args...)
}

func (d *DB) InsertGetID(query string, pkColumn string, args ...interface{}) (int64, error) {
	if d.DBType == "postgres" {
		var id int64
		q := d.PrepareQuery(query + " RETURNING " + pkColumn)
		err := d.DB.QueryRow(q, args...).Scan(&id)
		return id, err
	}

	res, err := d.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) migrate() error {
	dbType := os.Getenv("DB_TYPE")
	isPostgres := dbType == "postgres"

	pk := "INTEGER PRIMARY KEY AUTOINCREMENT"
	dt := "DATETIME"
	if isPostgres {
		pk = "SERIAL PRIMARY KEY"
		dt = "TIMESTAMP"
	}

	migrations := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
			id %s,
			telegram_id BIGINT UNIQUE NOT NULL,
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
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			updated_at %s DEFAULT CURRENT_TIMESTAMP
		)`, pk, dt, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS verification_codes (
			id %s,
			telegram_id BIGINT NOT NULL,
			email TEXT NOT NULL,
			code TEXT NOT NULL,
			expires_at %s NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			created_at %s DEFAULT CURRENT_TIMESTAMP
		)`, pk, dt, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS chat_sessions (
			id %s,
			user1_id BIGINT NOT NULL,
			user2_id BIGINT NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			started_at %s DEFAULT CURRENT_TIMESTAMP,
			ended_at %s,
			FOREIGN KEY (user1_id) REFERENCES users(telegram_id),
			FOREIGN KEY (user2_id) REFERENCES users(telegram_id)
		)`, pk, dt, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS chat_queue (
			id %s,
			telegram_id BIGINT UNIQUE NOT NULL,
			preferred_dept TEXT DEFAULT '',
			preferred_gender TEXT DEFAULT '',
			preferred_year INTEGER DEFAULT 0,
			joined_at %s DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS confessions (
			id %s,
			author_id BIGINT NOT NULL,
			content TEXT NOT NULL,
			department TEXT DEFAULT '',
			like_count INTEGER DEFAULT 0,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS confession_reactions (
			id %s,
			confession_id INTEGER NOT NULL,
			telegram_id BIGINT NOT NULL,
			reaction TEXT NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(confession_id, telegram_id),
			FOREIGN KEY (confession_id) REFERENCES confessions(id),
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS reports (
			id %s,
			reporter_id BIGINT NOT NULL,
			reported_id BIGINT NOT NULL,
			reason TEXT DEFAULT '',
			chat_session_id INTEGER DEFAULT 0,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (reporter_id) REFERENCES users(telegram_id),
			FOREIGN KEY (reported_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS blocked_users (
			id %s,
			user_id BIGINT NOT NULL,
			blocked_id BIGINT NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, blocked_id),
			FOREIGN KEY (user_id) REFERENCES users(telegram_id),
			FOREIGN KEY (blocked_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS whispers (
			id %s,
			sender_id BIGINT NOT NULL,
			target_dept TEXT NOT NULL,
			content TEXT NOT NULL,
			sender_dept TEXT DEFAULT '',
			sender_gender TEXT DEFAULT '',
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (sender_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS confession_replies (
			id %s,
			confession_id INTEGER NOT NULL,
			author_id BIGINT NOT NULL,
			content TEXT NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (confession_id) REFERENCES confessions(id),
			FOREIGN KEY (author_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS polls (
			id %s,
			author_id BIGINT NOT NULL,
			question TEXT NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS poll_options (
			id %s,
			poll_id INTEGER NOT NULL,
			option_text TEXT NOT NULL,
			vote_count INTEGER DEFAULT 0,
			FOREIGN KEY (poll_id) REFERENCES polls(id) ON DELETE CASCADE
		)`, pk),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS poll_votes (
			id %s,
			poll_id INTEGER NOT NULL,
			telegram_id BIGINT NOT NULL,
			option_id INTEGER NOT NULL,
			created_at %s DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(poll_id, telegram_id),
			FOREIGN KEY (poll_id) REFERENCES polls(id) ON DELETE CASCADE,
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS user_achievements (
			id %s,
			telegram_id BIGINT NOT NULL,
			achievement_key TEXT NOT NULL,
			earned_at %s DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(telegram_id, achievement_key),
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		`CREATE TABLE IF NOT EXISTS cs_messages (
			admin_message_id BIGINT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS cs_queue (
			user_id BIGINT PRIMARY KEY,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS cs_sessions (
			user_id BIGINT PRIMARY KEY,
			admin_id BIGINT NOT NULL,
			last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS rooms (
			id %s,
			slug TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT TRUE,
			created_at %s DEFAULT CURRENT_TIMESTAMP
		)`, pk, dt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS room_members (
			id %s,
			room_id INTEGER NOT NULL,
			telegram_id BIGINT NOT NULL,
			joined_at %s DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(room_id, telegram_id),
			FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
			FOREIGN KEY (telegram_id) REFERENCES users(telegram_id)
		)`, pk, dt),

		`CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_department ON users(department)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_active ON chat_sessions(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_queue_telegram_id ON chat_queue(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_confessions_created ON confessions(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_codes_telegram ON verification_codes(telegram_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_reported ON reports(reported_id)`,
	}

	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "already exists") ||
				strings.Contains(errStr, "duplicate key value") ||
				strings.Contains(errStr, "duplicate") {
				continue
			}
			return fmt.Errorf("migration failed: %s\nerror: %w", m, err)
		}
	}

	return nil
}
