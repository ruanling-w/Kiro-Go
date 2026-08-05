package config

// Tests for config location resolution. The high-stakes property is precedence:
// a repo checkout or Docker container that already keeps state in ./data must
// keep using it after the global-install support landed, or an upgrade looks
// like every account vanished.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPathPrecedence(t *testing.T) {
	// Isolate: chdir into an empty dir so no stray ./data interferes, and give
	// the process a fake HOME.
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // windows
	t.Setenv("CONFIG_PATH", "")

	// 4. no hints at all → home dir
	want := filepath.Join(home, AppDirName, "config.json")
	if got := ResolveConfigPath(""); got != want {
		t.Errorf("default = %q, want %q", got, want)
	}

	// 3. an existing ./data wins over the home dir — this is the upgrade path.
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ResolveConfigPath(""); got != filepath.Join("data", "config.json") {
		t.Errorf("with ./data = %q, want data/config.json", got)
	}

	// 2. CONFIG_PATH beats ./data (Docker sets it explicitly).
	t.Setenv("CONFIG_PATH", "/srv/kiro/config.json")
	if got := ResolveConfigPath(""); got != "/srv/kiro/config.json" {
		t.Errorf("CONFIG_PATH = %q", got)
	}

	// 1. --config beats everything.
	if got := ResolveConfigPath("/explicit/c.json"); got != "/explicit/c.json" {
		t.Errorf("explicit = %q", got)
	}
}

// A ./data/config.json without the directory listing (e.g. a bind-mounted file)
// must still be detected.
func TestResolveConfigPathFindsLooseDataFile(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	t.Setenv("HOME", filepath.Join(tmp, "home"))
	t.Setenv("CONFIG_PATH", "")

	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("data", "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ResolveConfigPath(""); got != filepath.Join("data", "config.json") {
		t.Fatalf("got %q, want data/config.json", got)
	}
}

// --host/--port must not be written into Config: the launcher picks a free port
// per run, and persisting it would silently relocate the server next time.
func TestRuntimeBindOverridesDoNotTouchConfig(t *testing.T) {
	// GetHost/GetPort read cfg, so a config must be loaded first.
	tmp := t.TempDir()
	if err := Init(filepath.Join(tmp, "config.json")); err != nil {
		t.Fatal(err)
	}

	cfgLock.Lock()
	prevHost, prevPort := bindHostOverride, bindPortOverride
	cfgLock.Unlock()
	t.Cleanup(func() {
		cfgLock.Lock()
		bindHostOverride, bindPortOverride = prevHost, prevPort
		cfgLock.Unlock()
	})

	configuredHost, configuredPort := GetHost(), GetPort()

	SetRuntimeBind("127.0.0.1", 9931)
	if got := GetBindHost(); got != "127.0.0.1" {
		t.Errorf("bind host = %q, want 127.0.0.1", got)
	}
	if got := GetBindPort(); got != 9931 {
		t.Errorf("bind port = %d, want 9931", got)
	}
	if GetHost() != configuredHost || GetPort() != configuredPort {
		t.Errorf("overrides leaked into config: host=%q port=%d", GetHost(), GetPort())
	}

	// Partial override: passing only a port must leave the host alone.
	cfgLock.Lock()
	bindHostOverride, bindPortOverride = "", 0
	cfgLock.Unlock()
	SetRuntimeBind("", 9932)
	if got := GetBindHost(); got != configuredHost {
		t.Errorf("host-only reset = %q, want configured %q", got, configuredHost)
	}
	if got := GetBindPort(); got != 9932 {
		t.Errorf("port = %d, want 9932", got)
	}
}

// The home fallback must be under the user's home, not a relative path that
// would land wherever the shell happened to be.
func TestResolveConfigPathHomeFallbackIsAbsolute(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("CONFIG_PATH", "")

	got := ResolveConfigPath("")
	if !filepath.IsAbs(got) {
		t.Fatalf("got %q, want absolute path", got)
	}
	if !strings.Contains(got, AppDirName) {
		t.Fatalf("got %q, want it under %s", got, AppDirName)
	}
}
