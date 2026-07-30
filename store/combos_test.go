package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

func testCombo() Combo {
	return Combo{ID: "c1", Name: "Primary", Strategy: "round-robin", StickyLimit: 2, FusionQuorum: 2, FusionTimeout: 1234, JudgeModel: "p/judge", Models: []ComboModel{{Model: "p/a"}, {Model: "p/b"}, {Model: "p/c"}}}
}

func TestComboNewDBCRUDOrderingAndCaseInsensitiveName(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "combo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	created, err := s.CreateCombo(testCombo())
	if err != nil {
		t.Fatal(err)
	}
	if created.Revision != 1 || created.Models[2].Position != 2 {
		t.Fatalf("created: %+v", created)
	}
	byName, err := s.GetComboByName("pRiMaRy")
	if err != nil {
		t.Fatal(err)
	}
	if byName.ID != "c1" || byName.Models[0].Model != "p/a" || byName.Models[2].Model != "p/c" {
		t.Fatalf("loaded: %+v", byName)
	}
	if _, err := s.CreateCombo(Combo{ID: "c2", Name: "PRIMARY"}); err == nil {
		t.Fatal("expected case-insensitive duplicate failure")
	}
	list, err := s.ListCombos()
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	created.Name = "Renamed"
	created.Models = []ComboModel{{Model: "x"}, {Model: "y"}}
	updated, err := s.UpdateCombo(created)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.Models[0].Model != "x" {
		t.Fatalf("update: %+v", updated)
	}
	if _, err = s.UpdateCombo(created); !errors.Is(err, ErrComboConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if err = s.DeleteCombo("missing", 1); !errors.Is(err, ErrComboNotFound) {
		t.Fatalf("not found: %v", err)
	}
	if err = s.DeleteCombo(updated.ID, 1); !errors.Is(err, ErrComboConflict) {
		t.Fatalf("delete conflict: %v", err)
	}
	if err = s.DeleteCombo(updated.ID, updated.Revision); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM combo_models`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("model cascade %d %v", n, err)
	}
}

func TestComboRepositoryValidationNeutralAndNilUnavailable(t *testing.T) {
	var nilStore *Store
	if _, err := nilStore.ListCombos(); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("nil list: %v", err)
	}
	if _, err := nilStore.CreateCombo(Combo{}); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("nil create: %v", err)
	}
	if err := nilStore.DeleteCombo("", 0); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("nil delete: %v", err)
	}
	s, err := Open(filepath.Join(t.TempDir(), "neutral.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := s.CreateCombo(Combo{ID: "", Name: "", Strategy: "anything", StickyLimit: -9, Models: []ComboModel{{Model: ""}}})
	if err != nil {
		t.Fatal(err)
	}
	if c.Strategy != "anything" || c.StickyLimit != -9 || len(c.Models) != 1 {
		t.Fatalf("repository altered values: %+v", c)
	}
}

func TestComboRotationStickyResetRestartAndCascade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rotate.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateCombo(testCombo())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"p/a", "p/a", "p/b"}
	for i, w := range want {
		got, err := s.ReserveComboRotation(c.ID, c.Revision)
		if err != nil || got[0].Model != w {
			t.Fatalf("reservation %d: %+v %v", i, got, err)
		}
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.ReserveComboRotation(c.ID, c.Revision)
	if err != nil || got[0].Model != "p/b" {
		t.Fatalf("restart: %+v %v", got, err)
	}
	c.Models = []ComboModel{{Model: "new/a"}, {Model: "new/b"}}
	c, err = s.UpdateCombo(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReserveComboRotation(c.ID, c.Revision-1); !errors.Is(err, ErrComboConflict) {
		t.Fatalf("old revision: %v", err)
	}
	got, err = s.ReserveComboRotation(c.ID, c.Revision)
	if err != nil || got[0].Model != "new/a" {
		t.Fatalf("reset: %+v %v", got, err)
	}
	if err = s.DeleteCombo(c.ID, c.Revision); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM combo_rotation`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rotation cascade: %d %v", n, err)
	}
}

func TestComboResetRotationIsRevisionBound(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "reset.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := s.CreateCombo(testCombo())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReserveComboRotation(c.ID, c.Revision); err != nil {
		t.Fatal(err)
	}
	c.Name = "Updated"
	c, err = s.UpdateCombo(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReserveComboRotation(c.ID, c.Revision); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetComboRotation(c.ID, c.Revision-1); !errors.Is(err, ErrComboConflict) {
		t.Fatalf("stale reset: %v", err)
	}
	var revision int64
	if err := s.db.QueryRow(`SELECT revision FROM combo_rotation WHERE combo_id=?`, c.ID).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != c.Revision {
		t.Fatalf("rotation revision=%d want %d", revision, c.Revision)
	}
	if err := s.ResetComboRotation(c.ID, c.Revision); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM combo_rotation WHERE combo_id=?`, c.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rotation count=%d err=%v", count, err)
	}
	if err := s.ResetComboRotation("missing", 1); !errors.Is(err, ErrComboNotFound) {
		t.Fatalf("missing reset: %v", err)
	}
}

func TestComboNameConflictIsTyped(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, err := s.CreateCombo(testCombo())
	if err != nil {
		t.Fatal(err)
	}
	duplicateID := testCombo()
	duplicateID.Name = "Different"
	if _, err := s.CreateCombo(duplicateID); err == nil || errors.Is(err, ErrComboNameConflict) {
		t.Fatalf("duplicate ID misclassified: %v", err)
	}

	second := testCombo()
	second.ID = "c2"
	second.Name = strings.ToUpper(first.Name)
	if _, err := s.CreateCombo(second); !errors.Is(err, ErrComboNameConflict) {
		t.Fatalf("create conflict: %v", err)
	}
	second.Name = "Secondary"
	second, err = s.CreateCombo(second)
	if err != nil {
		t.Fatal(err)
	}
	second.Name = first.Name
	if _, err := s.UpdateCombo(second); !errors.Is(err, ErrComboNameConflict) {
		t.Fatalf("update conflict: %v", err)
	}
}

func TestComboClosedStoreUnavailable(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "closed.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListCombos(); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("closed list: %v", err)
	}
	if _, err := s.CreateCombo(testCombo()); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("closed create: %v", err)
	}
	if err := s.ResetComboRotation("c1", 1); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("closed reset: %v", err)
	}
}

func TestComboRotationConcurrentAtomic(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c := testCombo()
	c.StickyLimit = 1
	c, err = s.CreateCombo(c)
	if err != nil {
		t.Fatal(err)
	}
	const count = 90
	hits := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := s.ReserveComboRotation(c.ID, c.Revision)
			if e != nil {
				errs <- e
				return
			}
			mu.Lock()
			hits[m[0].Model]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	for _, m := range c.Models {
		if hits[m.Model] != count/len(c.Models) {
			t.Fatalf("non-atomic distribution: %+v", hits)
		}
	}
}

func TestComboMigrationRecoversSeededSchemaWithoutTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seeded.db")
	db, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_version(version INTEGER NOT NULL); INSERT INTO schema_version VALUES(4)`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateCombo(testCombo()); err != nil {
		t.Fatalf("recovery create: %v", err)
	}
}

func TestComboMigrationFromV2PreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.db")
	db, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE schema_version(version INTEGER NOT NULL); INSERT INTO schema_version VALUES(2); CREATE TABLE request_logs(id INTEGER PRIMARY KEY AUTOINCREMENT,ts INTEGER NOT NULL,endpoint TEXT NOT NULL DEFAULT '',model TEXT NOT NULL DEFAULT '',account_id TEXT NOT NULL DEFAULT '',status TEXT NOT NULL DEFAULT '',error TEXT NOT NULL DEFAULT '',error_type TEXT NOT NULL DEFAULT '',tokens INTEGER NOT NULL DEFAULT 0,credits REAL NOT NULL DEFAULT 0,duration_ms INTEGER NOT NULL DEFAULT 0,client_ip TEXT NOT NULL DEFAULT '',api_key_id TEXT NOT NULL DEFAULT '',provider TEXT NOT NULL DEFAULT ''); INSERT INTO request_logs(ts,model) VALUES(7,'kept'); CREATE TABLE key_ip_stats(key_id TEXT NOT NULL,ip TEXT NOT NULL,requests INTEGER NOT NULL DEFAULT 0,first_seen INTEGER NOT NULL,last_seen INTEGER NOT NULL,PRIMARY KEY(key_id,ip)); INSERT INTO key_ip_stats VALUES('k','i',3,1,2)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var ver, n int
	if err = s.db.QueryRow(`SELECT version FROM schema_version`).Scan(&ver); err != nil || ver != schemaVersion {
		t.Fatalf("version %d %v", ver, err)
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM request_logs WHERE model='kept'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("logs lost")
	}
	if _, err = s.CreateCombo(testCombo()); err != nil {
		t.Fatal(err)
	}
}
