package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// RequestLogRow is the persisted form of an admin request log entry.
type RequestLogRow struct {
	Time      int64
	Endpoint  string
	Model     string
	AccountID string
	Status    string
	Error     string
	ErrorType string
	Tokens    int
	// Token breakdown (v6). Tokens stays = Input+Output for backward compatibility.
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	Cached              bool // response-cache hit (served without an upstream call)
	Credits             float64
	Duration            int64
	ClientIP            string
	ApiKeyID            string
	// Provider is the real upstream that served the request (kiro/grok/codex/...).
	// Admin-only; never exposed on the public check-key page.
	Provider        string
	RequestID       string
	ComboID         string
	ComboRevision   int64
	ComboStrategy   string
	CandidateModel  string
	EffectiveModel  string
	AttemptIndex    int
	FusionRole      string
	FailureClass    string
	BeforeFirstByte bool
	SelectedModel   string
	Billable        bool
}

// InsertRequestLogs appends rows in a single transaction. No-op on nil store or empty input.
func (s *Store) InsertRequestLogs(rows []RequestLogRow) error {
	if s == nil || s.db == nil || len(rows) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin insert logs: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
	INSERT INTO request_logs(
	  ts, endpoint, model, account_id, status, error, error_type,
	  tokens, credits, duration_ms, client_ip, api_key_id, provider,
	  request_id, combo_id, combo_revision, combo_strategy, candidate_model, effective_model,
	  attempt_index, fusion_role, failure_class, before_first_byte, selected_model, billable,
	  input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, cached
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("store: prepare insert logs: %w", err)
	}
	defer stmt.Close()

	rollup, err := tx.Prepare(`
		INSERT INTO usage_rollups(
		  provider, model, input_tokens, output_tokens, cache_read_tokens,
		  cache_creation_tokens, response_cache_hits, legacy_tokens,
		  detailed_rows, legacy_rows
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, model) DO UPDATE SET
		  input_tokens = input_tokens + excluded.input_tokens,
		  output_tokens = output_tokens + excluded.output_tokens,
		  cache_read_tokens = cache_read_tokens + excluded.cache_read_tokens,
		  cache_creation_tokens = cache_creation_tokens + excluded.cache_creation_tokens,
		  response_cache_hits = response_cache_hits + excluded.response_cache_hits,
		  legacy_tokens = legacy_tokens + excluded.legacy_tokens,
		  detailed_rows = detailed_rows + excluded.detailed_rows,
		  legacy_rows = legacy_rows + excluded.legacy_rows`)
	if err != nil {
		return fmt.Errorf("store: prepare usage rollup: %w", err)
	}
	defer rollup.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(
			r.Time, r.Endpoint, r.Model, r.AccountID, r.Status, r.Error, r.ErrorType,
			r.Tokens, r.Credits, r.Duration, r.ClientIP, r.ApiKeyID, r.Provider,
			r.RequestID, r.ComboID, r.ComboRevision, r.ComboStrategy, r.CandidateModel, r.EffectiveModel,
			r.AttemptIndex, r.FusionRole, r.FailureClass, r.BeforeFirstByte, r.SelectedModel, r.Billable,
			r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheCreationTokens, r.Cached,
		); err != nil {
			return fmt.Errorf("store: insert log: %w", err)
		}
		if r.Status == "success" && r.Endpoint != "combo_attempt" {
			detailed := r.InputTokens != 0 || r.OutputTokens != 0 || r.CacheReadTokens != 0 || r.CacheCreationTokens != 0
			legacyTokens, detailedRows, legacyRows := 0, 0, 0
			if detailed {
				detailedRows = 1
			} else if r.Tokens > 0 {
				legacyTokens, legacyRows = r.Tokens, 1
			}
			cached := 0
			if r.Cached {
				cached = 1
			}
			if _, err := rollup.Exec(
				strings.ToLower(strings.TrimSpace(r.Provider)), strings.TrimSpace(r.Model),
				r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheCreationTokens,
				cached, legacyTokens, detailedRows, legacyRows,
			); err != nil {
				return fmt.Errorf("store: update usage rollup: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit insert logs: %w", err)
	}
	return nil
}

// LoadRecentRequestLogs returns up to limit rows, oldest→newest (ready to seed a ring buffer).
// If limit <= 0, returns empty.
func (s *Store) LoadRecentRequestLogs(limit int) ([]RequestLogRow, error) {
	if s == nil || s.db == nil || limit <= 0 {
		return nil, nil
	}

	// Fetch newest first, then reverse for ring order.
	rows, err := s.db.Query(`
SELECT ts, endpoint, model, account_id, status, error, error_type,
       tokens, credits, duration_ms, client_ip, api_key_id, provider,
	       request_id, combo_id, combo_revision, combo_strategy, candidate_model, effective_model,
	       attempt_index, fusion_role, failure_class, before_first_byte, selected_model, billable,
	       input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, cached
FROM request_logs
ORDER BY id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: load recent logs: %w", err)
	}
	defer rows.Close()

	tmp := make([]RequestLogRow, 0, limit)
	for rows.Next() {
		var r RequestLogRow
		if err := rows.Scan(
			&r.Time, &r.Endpoint, &r.Model, &r.AccountID, &r.Status, &r.Error, &r.ErrorType,
			&r.Tokens, &r.Credits, &r.Duration, &r.ClientIP, &r.ApiKeyID, &r.Provider,
			&r.RequestID, &r.ComboID, &r.ComboRevision, &r.ComboStrategy, &r.CandidateModel, &r.EffectiveModel,
			&r.AttemptIndex, &r.FusionRole, &r.FailureClass, &r.BeforeFirstByte, &r.SelectedModel, &r.Billable,
			&r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheCreationTokens, &r.Cached,
		); err != nil {
			return nil, fmt.Errorf("store: scan log: %w", err)
		}
		tmp = append(tmp, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse to oldest→newest.
	out := make([]RequestLogRow, len(tmp))
	for i := range tmp {
		out[len(tmp)-1-i] = tmp[i]
	}
	return out, nil
}

// LoadRequestLogsByApiKeyID returns up to limit rows for one API key, newest→oldest.
// Reads the full SQLite history (not just the RAM ring buffer). Empty on nil store,
// empty apiKeyID, or limit <= 0.
func (s *Store) LoadRequestLogsByApiKeyID(apiKeyID string, limit int) ([]RequestLogRow, error) {
	if s == nil || s.db == nil || apiKeyID == "" || limit <= 0 {
		return nil, nil
	}

	rows, err := s.db.Query(`
SELECT ts, endpoint, model, account_id, status, error, error_type,
       tokens, credits, duration_ms, client_ip, api_key_id, provider,
	       request_id, combo_id, combo_revision, combo_strategy, candidate_model, effective_model,
	       attempt_index, fusion_role, failure_class, before_first_byte, selected_model, billable,
	       input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, cached
FROM request_logs
WHERE api_key_id = ?
ORDER BY id DESC
LIMIT ?`, apiKeyID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: load logs by api key: %w", err)
	}
	defer rows.Close()

	out := make([]RequestLogRow, 0, limit)
	for rows.Next() {
		var r RequestLogRow
		if err := rows.Scan(
			&r.Time, &r.Endpoint, &r.Model, &r.AccountID, &r.Status, &r.Error, &r.ErrorType,
			&r.Tokens, &r.Credits, &r.Duration, &r.ClientIP, &r.ApiKeyID, &r.Provider,
			&r.RequestID, &r.ComboID, &r.ComboRevision, &r.ComboStrategy, &r.CandidateModel, &r.EffectiveModel,
			&r.AttemptIndex, &r.FusionRole, &r.FailureClass, &r.BeforeFirstByte, &r.SelectedModel, &r.Billable,
			&r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheCreationTokens, &r.Cached,
		); err != nil {
			return nil, fmt.Errorf("store: scan log by api key: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// RequestLogAggregates contains detailed usage recoverable from persisted logs.
type RequestLogAggregates struct {
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	ResponseCacheHits   int64
	LegacyTokens        int64
	DetailedRows        int64
	LegacyRows          int64
}

// AggregateRequestLogUsage sums durable client-visible usage rollups.
func (s *Store) AggregateRequestLogUsage() (RequestLogAggregates, error) {
	var out RequestLogAggregates
	if s == nil || s.db == nil {
		return out, nil
	}
	err := s.db.QueryRow(`
	SELECT COALESCE(SUM(input_tokens), 0),
	       COALESCE(SUM(output_tokens), 0),
	       COALESCE(SUM(cache_read_tokens), 0),
	       COALESCE(SUM(cache_creation_tokens), 0),
	       COALESCE(SUM(response_cache_hits), 0),
	       COALESCE(SUM(legacy_tokens), 0),
	       COALESCE(SUM(detailed_rows), 0),
	       COALESCE(SUM(legacy_rows), 0)
	FROM usage_rollups`).Scan(
		&out.InputTokens,
		&out.OutputTokens,
		&out.CacheReadTokens,
		&out.CacheCreationTokens,
		&out.ResponseCacheHits,
		&out.LegacyTokens,
		&out.DetailedRows,
		&out.LegacyRows,
	)
	if err != nil && err != sql.ErrNoRows {
		return RequestLogAggregates{}, fmt.Errorf("store: aggregate request log usage: %w", err)
	}
	return out, nil
}

// ClearUsageRollups resets durable detailed usage without deleting request history.
func (s *Store) ClearUsageRollups() error {
	if s == nil || s.db == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.Exec(`DELETE FROM usage_rollups`); err != nil {
		return fmt.Errorf("store: clear usage rollups: %w", err)
	}
	return nil
}

// ClearRequestLogs deletes all persisted request logs.
func (s *Store) ClearRequestLogs() error {
	if s == nil || s.db == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.Exec(`DELETE FROM request_logs`); err != nil {
		return fmt.Errorf("store: clear logs: %w", err)
	}
	// Intentionally no VACUUM: not required for correctness and can fail under
	// active connections / locks depending on the SQLite driver.
	return nil
}

// PruneRequestLogs keeps only the newest maxRows. No-op if maxRows <= 0 or under cap.
func (s *Store) PruneRequestLogs(maxRows int) error {
	if s == nil || s.db == nil || maxRows <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM request_logs`).Scan(&n); err != nil {
		return fmt.Errorf("store: count logs: %w", err)
	}
	if n <= maxRows {
		return nil
	}
	// Delete everything older than the newest maxRows rows.
	_, err := s.db.Exec(`
DELETE FROM request_logs
WHERE id < (
  SELECT id FROM request_logs ORDER BY id DESC LIMIT 1 OFFSET ?
)`, maxRows-1)
	if err != nil {
		return fmt.Errorf("store: prune logs: %w", err)
	}
	return nil
}

// CountRequestLogs returns the number of persisted log rows (0 on nil store).
func (s *Store) CountRequestLogs() (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM request_logs`).Scan(&n)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return n, nil
}
