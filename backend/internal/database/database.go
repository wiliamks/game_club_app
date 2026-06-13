package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// InitDB initializes the SQLite database, runs migrations, and returns the DB connection handle
func InitDB() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./gamer_club.db"
	}

	// Self-healing: Ensure parent directory of database file exists with correct write permissions
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Printf("Warning: failed to create database parent directory %s: %v", dbDir, err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable Foreign Keys and WAL Mode for SQLite performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Add dynamic columns if upgrading an existing SQLite file
	_, _ = db.Exec("ALTER TABLE reviews ADD COLUMN title TEXT NOT NULL DEFAULT '';")
	_, _ = db.Exec("ALTER TABLE reviews ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN avatar_url TEXT;")
	_, _ = db.Exec("ALTER TABLE games ADD COLUMN time_to_beat_normal TEXT DEFAULT '';")
	_, _ = db.Exec("ALTER TABLE games ADD COLUMN time_to_beat_hastily TEXT DEFAULT '';")
	_, _ = db.Exec("ALTER TABLE games ADD COLUMN time_to_beat_completely TEXT DEFAULT '';")

	// Seed default admin user if database is empty
	if err := seedDefaultAdmin(db); err != nil {
		log.Printf("Warning: failed to seed default admin: %v", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
			avatar_url TEXT
		);`,

		`CREATE TABLE IF NOT EXISTS games (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			summary TEXT,
			cover_url TEXT,
			release_date DATETIME,
			time_to_beat TEXT,
			last_active_date DATETIME,
			is_active BOOLEAN NOT NULL DEFAULT 0,
			time_to_beat_normal TEXT DEFAULT '',
			time_to_beat_hastily TEXT DEFAULT '',
			time_to_beat_completely TEXT DEFAULT ''
		);`,

		`CREATE TABLE IF NOT EXISTS reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			username TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			gameplay INTEGER NOT NULL CHECK(gameplay >= 0 AND gameplay <= 5),
			art INTEGER NOT NULL CHECK(art >= 0 AND art <= 5),
			story INTEGER NOT NULL CHECK(story >= 0 AND story <= 5),
			soundtrack INTEGER NOT NULL CHECK(soundtrack >= 0 AND soundtrack <= 5),
			fun INTEGER NOT NULL CHECK(fun >= 0 AND fun <= 5),
			comment TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(game_id, user_id)
		);`,

		`CREATE TABLE IF NOT EXISTS voting_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			max_nominations INTEGER NOT NULL DEFAULT 3,
			phase TEXT NOT NULL CHECK(phase IN ('nomination', 'voting', 'closed')) DEFAULT 'nomination',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS nominations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES voting_sessions(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			game_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			cover_url TEXT,
			summary TEXT,
			UNIQUE(session_id, user_id, game_id)
		);`,

		`CREATE TABLE IF NOT EXISTS votes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES voting_sessions(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			preference TEXT NOT NULL, -- JSON array of GameIDs, e.g. "[123, 456]"
			UNIQUE(session_id, user_id)
		);`,
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			return fmt.Errorf("failed executing migration: %s, error: %w", q, err)
		}
	}
	return nil
}

func seedDefaultAdmin(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// No users exist, seed default admin: admin/admin
		hashed, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = db.Exec("INSERT INTO users (username, password, role) VALUES (?, ?, ?)", "admin", string(hashed), "admin")
		if err != nil {
			return err
		}
		log.Println("Database was empty. Seeded default admin account (username: 'admin', password: 'admin')")
	}

	return nil
}
