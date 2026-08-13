package store

import (
	"database/sql"
	"fmt"
)

// UsageRollup contains lifetime usage for one normalized provider/model pair.
type UsageRollup struct {
	Provider            string
	Model               string
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	LegacyTokens        int64
}

// ListUsageRollups returns durable lifetime usage grouped by provider and model.
func (s *Store) ListUsageRollups() ([]UsageRollup, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.Query(`
		SELECT provider, model, input_tokens, output_tokens,
		       cache_read_tokens, cache_creation_tokens, legacy_tokens
		FROM usage_rollups
		ORDER BY provider, model`)
	if err != nil {
		return nil, fmt.Errorf("store: list usage rollups: %w", err)
	}
	defer rows.Close()

	var out []UsageRollup
	for rows.Next() {
		var row UsageRollup
		if err := rows.Scan(
			&row.Provider,
			&row.Model,
			&row.InputTokens,
			&row.OutputTokens,
			&row.CacheReadTokens,
			&row.CacheCreationTokens,
			&row.LegacyTokens,
		); err != nil {
			return nil, fmt.Errorf("store: scan usage rollup: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("store: read usage rollups: %w", err)
	}
	return out, nil
}
