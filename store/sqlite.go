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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	schemaVersion = 8
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

	dsn := path + "?_pragma=foreign_keys(1)"
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
		// Seed at v1 so table creation and every version transition remain
		// recoverable if startup is interrupted before migration commits.
		if _, err := s.db.Exec(`INSERT INTO schema_version(version) VALUES (1)`); err != nil {
			return fmt.Errorf("store: seed schema_version: %w", err)
		}
		ver = 1
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

	// Combo schema reconciliation is atomic and runs on every startup. The
	// version row alone is not proof that an interrupted migration created every
	// table and column.
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
		  fusion_quorum INTEGER NOT NULL DEFAULT 0,
		  fusion_timeout_ms INTEGER NOT NULL DEFAULT 0,
		  judge_model TEXT NOT NULL DEFAULT '',
			  judge_provider TEXT NOT NULL DEFAULT '',
		  revision INTEGER NOT NULL DEFAULT 1,
		  created_at INTEGER NOT NULL,
		  updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS combo_models (
		  combo_id TEXT NOT NULL REFERENCES combos(id) ON DELETE CASCADE,
		  position INTEGER NOT NULL,
		  model TEXT NOT NULL,
		  provider TEXT NOT NULL DEFAULT '',
		  PRIMARY KEY (combo_id, position)
		)`,
		`CREATE TABLE IF NOT EXISTS combo_rotation (
		  combo_id TEXT PRIMARY KEY REFERENCES combos(id) ON DELETE CASCADE,
		  revision INTEGER NOT NULL,
		  model_index INTEGER NOT NULL DEFAULT 0,
		  use_count INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS chat_conversations (
		  id TEXT PRIMARY KEY, title TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '',
		  mode TEXT NOT NULL DEFAULT 'chat', status TEXT NOT NULL DEFAULT 'active', pinned INTEGER NOT NULL DEFAULT 0,
		  project_id TEXT, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, archived_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
		  id TEXT PRIMARY KEY, conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
		  parent_message_id TEXT REFERENCES chat_messages(id) ON DELETE SET NULL, client_request_id TEXT NOT NULL DEFAULT '',
		  role TEXT NOT NULL, content TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '',
		  status TEXT NOT NULL DEFAULT 'complete', error_code TEXT NOT NULL DEFAULT '', error_message TEXT NOT NULL DEFAULT '',
		  provider_response_id TEXT NOT NULL DEFAULT '', request_id TEXT NOT NULL DEFAULT '', input_tokens INTEGER NOT NULL DEFAULT 0,
		  output_tokens INTEGER NOT NULL DEFAULT 0, cache_read_tokens INTEGER NOT NULL DEFAULT 0, cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
		  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_attachments (
		  id TEXT PRIMARY KEY, conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
		  message_id TEXT REFERENCES chat_messages(id) ON DELETE CASCADE, kind TEXT NOT NULL, name TEXT NOT NULL DEFAULT '',
		  mime_type TEXT NOT NULL, size_bytes INTEGER NOT NULL, storage_key TEXT NOT NULL, width INTEGER, height INTEGER, created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_conversations_list ON chat_conversations(status, pinned DESC, updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_list ON chat_messages(conversation_id, created_at, id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_messages_client_request ON chat_messages(conversation_id, client_request_id) WHERE client_request_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_chat_attachments_conversation ON chat_attachments(conversation_id, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_attachments_message ON chat_attachments(message_id, created_at, id)`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("store: combo migration: %w", err)
		}
	}
	columns := map[string]bool{}
	rows, err := tx.Query(`PRAGMA table_info(combos)`)
	if err != nil {
		return fmt.Errorf("store: inspect combo columns: %w", err)
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return fmt.Errorf("store: scan combo columns: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("store: close combo columns: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: read combo columns: %w", err)
	}
	fusionColumns := []struct{ name, statement string }{
		{"fusion_quorum", `ALTER TABLE combos ADD COLUMN fusion_quorum INTEGER NOT NULL DEFAULT 0`},
		{"fusion_timeout_ms", `ALTER TABLE combos ADD COLUMN fusion_timeout_ms INTEGER NOT NULL DEFAULT 0`},
		{"judge_model", `ALTER TABLE combos ADD COLUMN judge_model TEXT NOT NULL DEFAULT ''`},
		{"judge_provider", `ALTER TABLE combos ADD COLUMN judge_provider TEXT NOT NULL DEFAULT ''`},
	}
	for _, column := range fusionColumns {
		if columns[column.name] {
			continue
		}
		if _, err := tx.Exec(column.statement); err != nil {
			return fmt.Errorf("store: add combo %s column: %w", column.name, err)
		}
	}
	modelColumns := map[string]bool{}
	modelRows, err := tx.Query(`PRAGMA table_info(combo_models)`)
	if err != nil {
		return fmt.Errorf("store: inspect combo model columns: %w", err)
	}
	for modelRows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := modelRows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			modelRows.Close()
			return fmt.Errorf("store: scan combo model columns: %w", err)
		}
		modelColumns[name] = true
	}
	if err := modelRows.Close(); err != nil {
		return fmt.Errorf("store: close combo model columns: %w", err)
	}
	if err := modelRows.Err(); err != nil {
		return fmt.Errorf("store: read combo model columns: %w", err)
	}
	if !modelColumns["provider"] {
		if _, err := tx.Exec(`ALTER TABLE combo_models ADD COLUMN provider TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("store: add combo model provider column: %w", err)
		}
	}
	// v5 adds backward-compatible per-attempt Combo observability columns.
	logColumns := []struct{ name, statement string }{
		{"request_id", `ALTER TABLE request_logs ADD COLUMN request_id TEXT NOT NULL DEFAULT ''`},
		{"combo_id", `ALTER TABLE request_logs ADD COLUMN combo_id TEXT NOT NULL DEFAULT ''`},
		{"combo_revision", `ALTER TABLE request_logs ADD COLUMN combo_revision INTEGER NOT NULL DEFAULT 0`},
		{"combo_strategy", `ALTER TABLE request_logs ADD COLUMN combo_strategy TEXT NOT NULL DEFAULT ''`},
		{"candidate_model", `ALTER TABLE request_logs ADD COLUMN candidate_model TEXT NOT NULL DEFAULT ''`},
		{"effective_model", `ALTER TABLE request_logs ADD COLUMN effective_model TEXT NOT NULL DEFAULT ''`},
		{"attempt_index", `ALTER TABLE request_logs ADD COLUMN attempt_index INTEGER NOT NULL DEFAULT 0`},
		{"fusion_role", `ALTER TABLE request_logs ADD COLUMN fusion_role TEXT NOT NULL DEFAULT ''`},
		{"failure_class", `ALTER TABLE request_logs ADD COLUMN failure_class TEXT NOT NULL DEFAULT ''`},
		{"before_first_byte", `ALTER TABLE request_logs ADD COLUMN before_first_byte INTEGER NOT NULL DEFAULT 0`},
		{"selected_model", `ALTER TABLE request_logs ADD COLUMN selected_model TEXT NOT NULL DEFAULT ''`},
		{"billable", `ALTER TABLE request_logs ADD COLUMN billable INTEGER NOT NULL DEFAULT 0`},
		// v6 splits the flat token count into input/output and adds prompt-cache
		// counts plus a response-cache-hit flag.
		{"input_tokens", `ALTER TABLE request_logs ADD COLUMN input_tokens INTEGER NOT NULL DEFAULT 0`},
		{"output_tokens", `ALTER TABLE request_logs ADD COLUMN output_tokens INTEGER NOT NULL DEFAULT 0`},
		{"cache_read_tokens", `ALTER TABLE request_logs ADD COLUMN cache_read_tokens INTEGER NOT NULL DEFAULT 0`},
		{"cache_creation_tokens", `ALTER TABLE request_logs ADD COLUMN cache_creation_tokens INTEGER NOT NULL DEFAULT 0`},
		{"cached", `ALTER TABLE request_logs ADD COLUMN cached INTEGER NOT NULL DEFAULT 0`},
	}
	logExisting := map[string]bool{}
	logRows, err := tx.Query(`PRAGMA table_info(request_logs)`)
	if err != nil {
		return fmt.Errorf("store: inspect request log columns: %w", err)
	}
	for logRows.Next() {
		var cid, notNull, primaryKey int
		var name, typ string
		var def any
		if err := logRows.Scan(&cid, &name, &typ, &notNull, &def, &primaryKey); err != nil {
			logRows.Close()
			return err
		}
		logExisting[name] = true
	}
	if err := logRows.Close(); err != nil {
		return err
	}
	for _, column := range logColumns {
		if !logExisting[column.name] {
			if _, err := tx.Exec(column.statement); err != nil {
				return fmt.Errorf("store: add request log %s column: %w", column.name, err)
			}
		}
	}
	if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, schemaVersion); err != nil {
		return fmt.Errorf("store: bump schema_version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit combo migration: %w", err)
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
