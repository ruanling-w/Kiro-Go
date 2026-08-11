package store

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOpenWindowsPath guards against the DSN regression where a native OS path
// (drive letter + backslashes on Windows) was wrapped in url.URL and rendered
// as file://D:%5C..., which SQLite could not open. t.TempDir() yields a
// backslash path on Windows, so this exercises the real failure mode there.
func TestOpenWindowsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kiro-runtime.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open native path: %v", err)
	}
	defer s.Close()
	if _, err := s.ListCombos(); err != nil {
		t.Fatalf("ListCombos after Open: %v", err)
	}
}

// TestOpenExistingEmptyFile covers the state left behind by a prior failed
// open: a 0-byte db file already on disk. Open must migrate it successfully
// rather than treating it as corrupt.
func TestOpenExistingEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kiro-runtime.db")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed empty file: %v", err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open empty file: %v", err)
	}
	defer s.Close()
	if _, err := s.ListCombos(); err != nil {
		t.Fatalf("ListCombos after Open: %v", err)
	}
}

func TestOpenMigrateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kiro-runtime.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	logs := []RequestLogRow{
		{Time: 100, Endpoint: "openai", Model: "m1", Status: "success", Tokens: 10, ClientIP: "1.1.1.1"},
		{Time: 200, Endpoint: "claude", Model: "m2", Status: "error", Error: "boom", ErrorType: "unknown", ClientIP: "2.2.2.2"},
		{Time: 300, Endpoint: "openai", Model: "m3", Status: "success", Tokens: 5, ApiKeyID: "k1"},
	}
	if err := s.InsertRequestLogs(logs); err != nil {
		t.Fatalf("InsertRequestLogs: %v", err)
	}

	loaded, err := s.LoadRecentRequestLogs(10)
	if err != nil {
		t.Fatalf("LoadRecentRequestLogs: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("want 3 logs, got %d", len(loaded))
	}
	// oldest → newest
	if loaded[0].Time != 100 || loaded[2].Time != 300 {
		t.Fatalf("order wrong: %+v", loaded)
	}
	if loaded[1].Error != "boom" || loaded[2].ApiKeyID != "k1" {
		t.Fatalf("fields mismatch: %+v", loaded)
	}

	// reopen
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	loaded2, err := s2.LoadRecentRequestLogs(2)
	if err != nil {
		t.Fatalf("LoadRecent after reopen: %v", err)
	}
	if len(loaded2) != 2 || loaded2[0].Time != 200 || loaded2[1].Time != 300 {
		t.Fatalf("limit/order after reopen: %+v", loaded2)
	}
}

func TestAggregateRequestLogUsageSeparatesLegacyTokens(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "usage.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	rows := []RequestLogRow{
		{Time: 1, Endpoint: "openai", Status: "success", Tokens: 150, InputTokens: 120, OutputTokens: 30, CacheReadTokens: 80, CacheCreationTokens: 10},
		{Time: 2, Endpoint: "claude", Status: "success", Tokens: 75},
		{Time: 3, Endpoint: "openai", Status: "error", Tokens: 25},
		{Time: 4, Endpoint: "combo_attempt", Status: "success", Tokens: 50, InputTokens: 40, OutputTokens: 10},
	}
	if err := s.InsertRequestLogs(rows); err != nil {
		t.Fatal(err)
	}

	got, err := s.AggregateRequestLogUsage()
	if err != nil {
		t.Fatal(err)
	}
	if got.InputTokens != 120 || got.OutputTokens != 30 || got.CacheReadTokens != 80 || got.CacheCreationTokens != 10 {
		t.Fatalf("breakdown=%+v", got)
	}
	if got.LegacyTokens != 75 || got.DetailedRows != 1 || got.LegacyRows != 1 {
		t.Fatalf("coverage=%+v", got)
	}
}

func TestPruneAndClearLogs(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	rows := make([]RequestLogRow, 0, 20)
	for i := 1; i <= 20; i++ {
		rows = append(rows, RequestLogRow{Time: int64(i), Endpoint: "x", Status: "success"})
	}
	if err := s.InsertRequestLogs(rows); err != nil {
		t.Fatal(err)
	}
	if err := s.PruneRequestLogs(5); err != nil {
		t.Fatal(err)
	}
	n, err := s.CountRequestLogs()
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("after prune want 5, got %d", n)
	}
	kept, err := s.LoadRecentRequestLogs(10)
	if err != nil {
		t.Fatal(err)
	}
	if kept[0].Time != 16 || kept[4].Time != 20 {
		t.Fatalf("prune kept wrong rows: %+v", kept)
	}

	if err := s.ClearRequestLogs(); err != nil {
		t.Fatal(err)
	}
	n, _ = s.CountRequestLogs()
	if n != 0 {
		t.Fatalf("clear want 0, got %d", n)
	}
}

func TestKeyIPUpsertLoadDelete(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "ip.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.UpsertKeyIPStats([]KeyIPRow{
		{KeyID: "k1", IP: "1.1.1.1", Requests: 5, FirstSeen: 10, LastSeen: 20},
		{KeyID: "k1", IP: "2.2.2.2", Requests: 1, FirstSeen: 11, LastSeen: 21},
		{KeyID: "k2", IP: "3.3.3.3", Requests: 9, FirstSeen: 1, LastSeen: 2},
	}); err != nil {
		t.Fatal(err)
	}

	// bump k1/1.1.1.1 — first_seen should stay 10
	if err := s.UpsertKeyIPStats([]KeyIPRow{
		{KeyID: "k1", IP: "1.1.1.1", Requests: 8, FirstSeen: 15, LastSeen: 30},
	}); err != nil {
		t.Fatal(err)
	}

	all, err := s.LoadKeyIPStats()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || len(all["k1"]) != 2 {
		t.Fatalf("unexpected map: %+v", all)
	}
	r := all["k1"]["1.1.1.1"]
	if r.Requests != 8 || r.FirstSeen != 10 || r.LastSeen != 30 {
		t.Fatalf("upsert merge wrong: %+v", r)
	}

	if err := s.DeleteKeyIPStats("k1"); err != nil {
		t.Fatal(err)
	}
	all, _ = s.LoadKeyIPStats()
	if _, ok := all["k1"]; ok {
		t.Fatalf("k1 should be gone: %+v", all)
	}
	if len(all["k2"]) != 1 {
		t.Fatalf("k2 should remain: %+v", all)
	}
}

func TestRequestLogComboAttemptRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "attempt.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	want := RequestLogRow{Time: 1, Endpoint: "openai", Model: "requested-combo", AccountID: "acct", Status: "success", Provider: "kiro", RequestID: "req-1", ComboID: "combo-1", ComboRevision: 7, ComboStrategy: "fusion", CandidateModel: "candidate", EffectiveModel: "effective", AttemptIndex: 3, FusionRole: "judge", FailureClass: "", BeforeFirstByte: true, SelectedModel: "effective", Billable: true, Tokens: 42, Credits: 1.25}
	if err := s.InsertRequestLogs([]RequestLogRow{want}); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadRecentRequestLogs(1)
	if err != nil || len(got) != 1 {
		t.Fatalf("load attempt: len=%d err=%v", len(got), err)
	}
	if got[0] != want {
		t.Fatalf("attempt mismatch:\n got %+v\nwant %+v", got[0], want)
	}
}

func TestRequestLogTokenBreakdownRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "tokens.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	want := RequestLogRow{
		Time: 1, Endpoint: "claude", Model: "claude-opus-4.8", AccountID: "acct",
		Status: "success", Provider: "kiro", Tokens: 300, Credits: 2.5,
		InputTokens: 200, OutputTokens: 100, CacheReadTokens: 150, CacheCreationTokens: 40, Cached: true,
	}
	if err := s.InsertRequestLogs([]RequestLogRow{want}); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadRecentRequestLogs(1)
	if err != nil || len(got) != 1 {
		t.Fatalf("load: len=%d err=%v", len(got), err)
	}
	if got[0] != want {
		t.Fatalf("token breakdown mismatch:\n got %+v\nwant %+v", got[0], want)
	}
}

func TestStoreCloseIdempotentConcurrent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "close.db"))
	if err != nil {
		t.Fatal(err)
	}
	const callers = 32
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() { errs <- s.Close() }()
	}
	for i := 0; i < callers; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("repeated Close: %v", err)
	}
}

func TestNilStoreSafe(t *testing.T) {
	var s *Store
	if err := s.InsertRequestLogs([]RequestLogRow{{Time: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadRecentRequestLogs(10); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearRequestLogs(); err != nil {
		t.Fatal(err)
	}
	if err := s.PruneRequestLogs(1); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertKeyIPStats([]KeyIPRow{{KeyID: "k", IP: "1", Requests: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadKeyIPStats(); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteKeyIPStats("k"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePath(t *testing.T) {
	t.Setenv("RUNTIME_DB_PATH", "")
	p := ResolvePath("/app/data")
	if p != filepath.Join("/app/data", "kiro-runtime.db") {
		t.Fatalf("default path: %s", p)
	}
	t.Setenv("RUNTIME_DB_PATH", "/tmp/custom.db")
	if ResolvePath("/app/data") != "/tmp/custom.db" {
		t.Fatal("env override failed")
	}
}
