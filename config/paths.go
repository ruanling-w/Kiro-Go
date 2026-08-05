package config

// paths.go decides where mutable state lives. Kiro-Go is shipped two ways and
// they want different answers:
//
//   - Repo checkout / Docker image: state belongs next to the code in ./data,
//     which is also the mount point (VOLUME /app/data).
//   - Global `npm i -g kiroproxy` install: there is no checkout and the CWD is
//     wherever the user happened to be, so state belongs in the home directory.
//
// Resolution is ordered so that an existing ./data keeps winning — upgrading an
// existing deployment must never silently start from a blank config and appear
// to have lost every account.

import (
	"os"
	"path/filepath"
)

// AppDirName is the per-user state directory created under $HOME.
const AppDirName = ".kiroproxy"

// legacyDataDir is the in-repo state directory used before the global install
// existed. Still preferred whenever it is already present.
const legacyDataDir = "data"

// ResolveConfigPath returns the config.json path to use, in priority order:
//
//  1. explicit is non-empty (the --config flag)
//  2. $CONFIG_PATH
//  3. ./data/config.json — when that file or directory already exists
//  4. $HOME/.kiroproxy/config.json
//
// The last case also applies when the home directory cannot be determined, in
// which case it degrades to ./data/config.json rather than failing.
func ResolveConfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("CONFIG_PATH"); env != "" {
		return env
	}

	legacy := filepath.Join(legacyDataDir, "config.json")
	if pathExists(legacy) || pathExists(legacyDataDir) {
		return legacy
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return legacy
	}
	return filepath.Join(home, AppDirName, "config.json")
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// Runtime bind overrides come from --host/--port. They are deliberately kept
// out of Config: the CLI launcher picks a different port when the configured one
// is busy, and persisting that ephemeral choice into config.json would silently
// move the server for every later run. GetPort/GetHost keep reporting the
// configured values (that is what the admin settings form edits); only the
// actual listener consults these.
var (
	bindHostOverride string
	bindPortOverride int
)

// SetRuntimeBind records --host/--port. Empty host and zero port are ignored, so
// passing only one flag leaves the other on its configured value.
func SetRuntimeBind(host string, port int) {
	cfgLock.Lock()
	defer cfgLock.Unlock()
	if host != "" {
		bindHostOverride = host
	}
	if port > 0 {
		bindPortOverride = port
	}
}

// GetBindHost returns the host the listener should bind to.
func GetBindHost() string {
	cfgLock.RLock()
	override := bindHostOverride
	cfgLock.RUnlock()
	if override != "" {
		return override
	}
	return GetHost()
}

// GetBindPort returns the port the listener should bind to.
func GetBindPort() int {
	cfgLock.RLock()
	override := bindPortOverride
	cfgLock.RUnlock()
	if override > 0 {
		return override
	}
	return GetPort()
}
