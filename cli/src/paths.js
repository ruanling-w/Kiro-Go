"use strict";

// paths.js — where the launcher keeps per-user state.
//
// Must agree with config/paths.go (AppDirName = ".kiroproxy"): the CLI passes
// --config explicitly, but the Go side falls back to the same directory when
// run directly, and the two disagreeing would silently split a user's accounts
// across two config files.

const os = require("os");
const path = require("path");

const APP_DIR_NAME = ".kiroproxy";

function appDir() {
  return path.join(os.homedir(), APP_DIR_NAME);
}

function binDir() {
  return path.join(appDir(), "bin");
}

// Windows needs the .exe suffix for spawn to find the file.
function binaryName() {
  return process.platform === "win32" ? "kiro-go.exe" : "kiro-go";
}

function binaryPath() {
  return path.join(binDir(), binaryName());
}

// Records which release the installed binary came from, so `npm update -g`
// bumping the package version triggers a re-download instead of silently
// running the old server.
function versionStampPath() {
  return path.join(binDir(), ".version");
}

function configPath() {
  return path.join(appDir(), "config.json");
}

module.exports = {
  APP_DIR_NAME,
  appDir,
  binDir,
  binaryName,
  binaryPath,
  versionStampPath,
  configPath,
};
