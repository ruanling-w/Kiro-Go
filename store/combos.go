package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"modernc.org/sqlite"
)

var (
	ErrStorageUnavailable = errors.New("store: storage unavailable")
	ErrComboNotFound      = errors.New("store: combo not found")
	ErrComboConflict      = errors.New("store: combo revision conflict")
	ErrComboNameConflict  = errors.New("store: combo name conflict")
)

// Combo is persisted configuration. Repository methods deliberately do not
// impose application-level validation on its values.
type Combo struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Strategy      string       `json:"strategy"`
	StickyLimit   int          `json:"stickyLimit"`
	FusionQuorum  int          `json:"fusionQuorum,omitempty"`
	FusionTimeout int          `json:"fusionTimeoutMs,omitempty"`
	JudgeModel    string       `json:"judgeModel,omitempty"`
	Revision      int64        `json:"revision"`
	CreatedAt     int64        `json:"createdAt"`
	UpdatedAt     int64        `json:"updatedAt"`
	Models        []ComboModel `json:"models"`
}

// ComboModel is one model in a combo. Position is normalized on writes and
// returned in ascending order.
type ComboModel struct {
	Model    string `json:"model"`
	Position int    `json:"position"`
}

func (s *Store) comboDBLocked() (*sql.DB, error) {
	if s == nil || s.db == nil {
		return nil, ErrStorageUnavailable
	}
	return s.db, nil
}

func scanCombo(row interface{ Scan(...any) error }) (Combo, error) {
	var c Combo
	err := row.Scan(&c.ID, &c.Name, &c.Strategy, &c.StickyLimit, &c.FusionQuorum, &c.FusionTimeout, &c.JudgeModel, &c.Revision, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func loadComboModels(q interface {
	Query(string, ...any) (*sql.Rows, error)
}, c *Combo) error {
	rows, err := q.Query(`SELECT model, position FROM combo_models WHERE combo_id = ? ORDER BY position`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	c.Models = []ComboModel{}
	for rows.Next() {
		var m ComboModel
		if err := rows.Scan(&m.Model, &m.Position); err != nil {
			return err
		}
		c.Models = append(c.Models, m)
	}
	return rows.Err()
}

func insertComboModels(tx *sql.Tx, id string, models []ComboModel) error {
	stmt, err := tx.Prepare(`INSERT INTO combo_models(combo_id, position, model) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, m := range models {
		if _, err := stmt.Exec(id, i, m.Model); err != nil {
			return err
		}
	}
	return nil
}

func isComboNameConflict(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	// SQLite extended code 2067 is SQLITE_CONSTRAINT_UNIQUE. The error text
	// identifies the name index; primary-key and unrelated unique constraints
	// must not be reported as a name conflict.
	return sqliteErr.Code() == 2067 && strings.Contains(sqliteErr.Error(), "combos.name")
}

// CreateCombo inserts a combo and its ordered models atomically.
func (s *Store) CreateCombo(c Combo) (Combo, error) {
	if s == nil {
		return Combo{}, ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return Combo{}, err
	}
	tx, err := db.Begin()
	if err != nil {
		return Combo{}, fmt.Errorf("store: begin create combo: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UnixMilli()
	c.Revision = 1
	c.CreatedAt, c.UpdatedAt = now, now
	if _, err := tx.Exec(`INSERT INTO combos(id,name,strategy,sticky_limit,fusion_quorum,fusion_timeout_ms,judge_model,revision,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, c.ID, c.Name, c.Strategy, c.StickyLimit, c.FusionQuorum, c.FusionTimeout, c.JudgeModel, c.Revision, now, now); err != nil {
		if isComboNameConflict(err) {
			return Combo{}, ErrComboNameConflict
		}
		return Combo{}, fmt.Errorf("store: create combo: %w", err)
	}
	if err := insertComboModels(tx, c.ID, c.Models); err != nil {
		return Combo{}, fmt.Errorf("store: create combo models: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Combo{}, fmt.Errorf("store: commit create combo: %w", err)
	}
	for i := range c.Models {
		c.Models[i].Position = i
	}
	return c, nil
}

// GetCombo returns a combo by ID.
func (s *Store) GetCombo(id string) (Combo, error) {
	if s == nil {
		return Combo{}, ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return Combo{}, err
	}
	c, err := scanCombo(db.QueryRow(`SELECT id,name,strategy,sticky_limit,fusion_quorum,fusion_timeout_ms,judge_model,revision,created_at,updated_at FROM combos WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return Combo{}, ErrComboNotFound
	}
	if err != nil {
		return Combo{}, fmt.Errorf("store: get combo: %w", err)
	}
	if err := loadComboModels(db, &c); err != nil {
		return Combo{}, fmt.Errorf("store: get combo models: %w", err)
	}
	return c, nil
}

// GetComboByName performs a case-insensitive lookup.
func (s *Store) GetComboByName(name string) (Combo, error) {
	if s == nil {
		return Combo{}, ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return Combo{}, err
	}
	c, err := scanCombo(db.QueryRow(`SELECT id,name,strategy,sticky_limit,fusion_quorum,fusion_timeout_ms,judge_model,revision,created_at,updated_at FROM combos WHERE name=? COLLATE NOCASE`, name))
	if err == sql.ErrNoRows {
		return Combo{}, ErrComboNotFound
	}
	if err != nil {
		return Combo{}, fmt.Errorf("store: get combo by name: %w", err)
	}
	if err := loadComboModels(db, &c); err != nil {
		return Combo{}, fmt.Errorf("store: get combo models: %w", err)
	}
	return c, nil
}

// ListCombos returns combos in creation order with model order preserved.
func (s *Store) ListCombos() ([]Combo, error) {
	if s == nil {
		return nil, ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id,name,strategy,sticky_limit,fusion_quorum,fusion_timeout_ms,judge_model,revision,created_at,updated_at FROM combos ORDER BY created_at,id`)
	if err != nil {
		return nil, fmt.Errorf("store: list combos: %w", err)
	}
	var out []Combo
	for rows.Next() {
		c, err := scanCombo(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := loadComboModels(db, &out[i]); err != nil {
			return nil, fmt.Errorf("store: list combo models: %w", err)
		}
	}
	return out, nil
}

// UpdateCombo replaces mutable fields when c.Revision matches. Any successful
// update advances the revision and resets rotation state.
func (s *Store) UpdateCombo(c Combo) (Combo, error) {
	if s == nil {
		return Combo{}, ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return Combo{}, err
	}
	tx, err := db.Begin()
	if err != nil {
		return Combo{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var existing int64
	if err := tx.QueryRow(`SELECT revision FROM combos WHERE id=?`, c.ID).Scan(&existing); err == sql.ErrNoRows {
		return Combo{}, ErrComboNotFound
	} else if err != nil {
		return Combo{}, err
	}
	if existing != c.Revision {
		return Combo{}, ErrComboConflict
	}
	now := time.Now().UnixMilli()
	res, err := tx.Exec(`UPDATE combos SET name=?,strategy=?,sticky_limit=?,fusion_quorum=?,fusion_timeout_ms=?,judge_model=?,revision=revision+1,updated_at=? WHERE id=? AND revision=?`, c.Name, c.Strategy, c.StickyLimit, c.FusionQuorum, c.FusionTimeout, c.JudgeModel, now, c.ID, c.Revision)
	if err != nil {
		if isComboNameConflict(err) {
			return Combo{}, ErrComboNameConflict
		}
		return Combo{}, fmt.Errorf("store: update combo: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return Combo{}, ErrComboConflict
	}
	if _, err := tx.Exec(`DELETE FROM combo_models WHERE combo_id=?`, c.ID); err != nil {
		return Combo{}, err
	}
	if err := insertComboModels(tx, c.ID, c.Models); err != nil {
		return Combo{}, err
	}
	if _, err := tx.Exec(`DELETE FROM combo_rotation WHERE combo_id=?`, c.ID); err != nil {
		return Combo{}, err
	}
	if err := tx.Commit(); err != nil {
		return Combo{}, err
	}
	c.Revision++
	c.UpdatedAt = now
	for i := range c.Models {
		c.Models[i].Position = i
	}
	return c, nil
}

// DeleteCombo deletes only the requested revision. Foreign-key cascades remove
// models and rotation state.
func (s *Store) DeleteCombo(id string, revision int64) error {
	if s == nil {
		return ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return err
	}
	var current int64
	if err := db.QueryRow(`SELECT revision FROM combos WHERE id=?`, id).Scan(&current); err == sql.ErrNoRows {
		return ErrComboNotFound
	} else if err != nil {
		return err
	}
	if current != revision {
		return ErrComboConflict
	}
	res, err := db.Exec(`DELETE FROM combos WHERE id=? AND revision=?`, id, revision)
	if err != nil {
		return fmt.Errorf("store: delete combo: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrComboConflict
	}
	return nil
}

// ResetComboRotation clears persistent rotation state for the requested revision.
func (s *Store) ResetComboRotation(id string, revision int64) error {
	if s == nil {
		return ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var current int64
	if err := tx.QueryRow(`SELECT revision FROM combos WHERE id=?`, id).Scan(&current); err == sql.ErrNoRows {
		return ErrComboNotFound
	} else if err != nil {
		return err
	}
	if current != revision {
		return ErrComboConflict
	}
	if _, err := tx.Exec(`DELETE FROM combo_rotation WHERE combo_id=? AND revision=?`, id, revision); err != nil {
		return err
	}
	return tx.Commit()
}

// ReserveComboRotation atomically reserves and returns the models in routing
// order for one request. State advances after StickyLimit reservations and is
// tied to the supplied configuration revision.
func (s *Store) ReserveComboRotation(id string, revision int64) ([]ComboModel, error) {
	if s == nil {
		return nil, ErrStorageUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db, err := s.comboDBLocked()
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var sticky int
	var current int64
	if err := tx.QueryRow(`SELECT sticky_limit,revision FROM combos WHERE id=?`, id).Scan(&sticky, &current); err == sql.ErrNoRows {
		return nil, ErrComboNotFound
	} else if err != nil {
		return nil, err
	}
	if current != revision {
		return nil, ErrComboConflict
	}
	var models []ComboModel
	rows, err := tx.Query(`SELECT model,position FROM combo_models WHERE combo_id=? ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m ComboModel
		if err := rows.Scan(&m.Model, &m.Position); err != nil {
			rows.Close()
			return nil, err
		}
		models = append(models, m)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(models) <= 1 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return models, nil
	}
	if sticky < 1 {
		sticky = 1
	}
	var idx, count int
	err = tx.QueryRow(`SELECT model_index,use_count FROM combo_rotation WHERE combo_id=? AND revision=?`, id, revision).Scan(&idx, &count)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		idx, count = 0, 0
	}
	idx = ((idx % len(models)) + len(models)) % len(models)
	ordered := append(append([]ComboModel{}, models[idx:]...), models[:idx]...)
	count++
	if count >= sticky {
		idx = (idx + 1) % len(models)
		count = 0
	}
	_, err = tx.Exec(`INSERT INTO combo_rotation(combo_id,revision,model_index,use_count) VALUES(?,?,?,?) ON CONFLICT(combo_id) DO UPDATE SET revision=excluded.revision,model_index=excluded.model_index,use_count=excluded.use_count`, id, revision, idx, count)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ordered, nil
}
