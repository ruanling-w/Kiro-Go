// Package store provides a SQLite-backed runtime store for data that should
// survive process restarts without bloating config.json:
//   - request_logs (admin Logs page)
//   - key_ip_stats (per-API-key client IP lifetime counters)
//
// Hot-path request handling must not call this package synchronously for every
// request; callers batch writes via a background flusher. Open failures are
// non-fatal: a nil *Store is safe and all methods no-op / return empty data.
package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	schemaVersion = 3
	driverName    = "sqlite"
)

// Store is a thin SQLite wrapper. Methods are safe for concurrent use.
// A nil receiver is treated as "disabled" (fail-open).
type Store struct {
	db   *sql.DB
	path string
	mu   sync.Mutex // serialize writes; single-writer friendliness
}

// DefaultPath returns {configDir}/kiro-runtime.db.
func DefaultPath(configDir string) string {
	if configDir == "" {
		configDir = "."
	}
	return filepath.Join(configDir, "kiro-runtime.db")
}

// ResolvePath returns RUNTIME_DB_PATH if set, otherwise DefaultPath(configDir).
func ResolvePath(configDir string) string {
	if p := os.Getenv("RUNTIME_DB_PATH"); p != "" {
		return p
	}
	return DefaultPath(configDir)
}

// Open opens (or creates) the runtime DB at path, applies pragmas and migrations.
// On any error it returns (nil, err) so callers can fail-open.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("store: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("store: mkdir: %w", err)
	}

	// Ensure file is created with restricted perms when new.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, createErr := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
		if createErr != nil {
			return nil, fmt.Errorf("store: create: %w", createErr)
		}
		_ = f.Close()
	}

	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "_pragma=foreign_keys(1)"}).String()
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	// Keep SQLite simple under WAL. The DSN applies foreign_keys to every new
	// connection, including replacements created after lifetime recycling.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s := &Store{db: db, path: path}
	if err := s.applyPragmas(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) applyPragmas() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := s.db.Exec(p); err != nil {
			return fmt.Errorf("store: %s: %w", p, err)
		}
	}
	return nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL
)`); err != nil {
		return fmt.Errorf("store: schema_version table: %w", err)
	}

	var ver int
	err := s.db.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&ver)
	if err == sql.ErrNoRows {
		if _, err := s.db.Exec(`INSERT INTO schema_version(version) VALUES (?)`, schemaVersion); err != nil {
			return fmt.Errorf("store: seed schema_version: %w", err)
		}
		ver = schemaVersion
	} else if err != nil {
		return fmt.Errorf("store: read schema_version: %w", err)
	}

	if ver > schemaVersion {
		return fmt.Errorf("store: database schema version %d is newer than supported %d", ver, schemaVersion)
	}

	// v1 tables (idempotent).
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS request_logs (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  ts           INTEGER NOT NULL,
  endpoint     TEXT    NOT NULL DEFAULT '',
  model        TEXT    NOT NULL DEFAULT '',
  account_id   TEXT    NOT NULL DEFAULT '',
  status       TEXT    NOT NULL DEFAULT '',
  error        TEXT    NOT NULL DEFAULT '',
  error_type   TEXT    NOT NULL DEFAULT '',
  tokens       INTEGER NOT NULL DEFAULT 0,
  credits      REAL    NOT NULL DEFAULT 0,
  duration_ms  INTEGER NOT NULL DEFAULT 0,
  client_ip    TEXT    NOT NULL DEFAULT '',
  api_key_id   TEXT    NOT NULL DEFAULT '',
  provider     TEXT    NOT NULL DEFAULT ''
)`); err != nil {
		return fmt.Errorf("store: request_logs: %w", err)
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_request_logs_ts ON request_logs(ts DESC)`); err != nil {
		return fmt.Errorf("store: idx_request_logs_ts: %w", err)
	}

	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS key_ip_stats (
  key_id     TEXT    NOT NULL,
  ip         TEXT    NOT NULL,
  requests   INTEGER NOT NULL DEFAULT 0,
  first_seen INTEGER NOT NULL,
  last_seen  INTEGER NOT NULL,
  PRIMARY KEY (key_id, ip)
)`); err != nil {
		return fmt.Errorf("store: key_ip_stats: %w", err)
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_key_ip_last ON key_ip_stats(key_id, last_seen DESC)`); err != nil {
		return fmt.Errorf("store: idx_key_ip_last: %w", err)
	}

	// v2: admin-only real provider column on request_logs.
	if ver < 2 {
		if _, err := s.db.Exec(`ALTER TABLE request_logs ADD COLUMN provider TEXT NOT NULL DEFAULT ''`); err != nil {
			// Ignore "duplicate column" if a previous partial migration ran.
			if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				return fmt.Errorf("store: add provider column: %w", err)
			}
		}
	}

	// v3: normalized combo configuration and persistent round-robin state.
	if ver <= 3 {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("store: begin combo migration: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		statements := []string{
			`CREATE TABLE IF NOT EXISTS combos (
			  id TEXT PRIMARY KEY,
			  name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			  strategy TEXT NOT NULL DEFAULT '',
			  sticky_limit INTEGER NOT NULL DEFAULT 1,
			  revision INTEGER NOT NULL DEFAULT 1,
			  created_at INTEGER NOT NULL,
			  updated_at INTEGER NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS combo_models (
			  combo_id TEXT NOT NULL REFERENCES combos(id) ON DELETE CASCADE,
			  position INTEGER NOT NULL,
			  model TEXT NOT NULL,
			  PRIMARY KEY (combo_id, position)
			)`,
			`CREATE TABLE IF NOT EXISTS combo_rotation (
			  combo_id TEXT PRIMARY KEY REFERENCES combos(id) ON DELETE CASCADE,
			  revision INTEGER NOT NULL,
			  model_index INTEGER NOT NULL DEFAULT 0,
			  use_count INTEGER NOT NULL DEFAULT 0
			)`,
		}
		for _, statement := range statements {
			if _, err := tx.Exec(statement); err != nil {
				return fmt.Errorf("store: combo migration: %w", err)
			}
		}
		if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, schemaVersion); err != nil {
			return fmt.Errorf("store: bump schema_version: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: commit combo migration: %w", err)
		}
	}
	return nil
}

// Path returns the on-disk path, or "" if s is nil.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Close closes the underlying DB. Safe on nil.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
