"use strict";

// downloadBinary.js — fetch the Go server binary for this machine from the
// GitHub release matching the installed package version.
//
// Why download instead of bundling: the binary embeds the admin SPA and is
// ~22 MB per platform. Publishing six of them inside one npm tarball would make
// every install pay for five architectures it cannot run.
//
// Integrity: every release ships a SHA256SUMS file. A binary whose digest does
// not match is discarded, never executed — this is the only thing standing
// between a compromised CDN response and arbitrary code running as the user.

const fs = require("fs");
const path = require("path");
const https = require("https");
const crypto = require("crypto");
const { URL } = require("url");

const { binDir, binaryPath, binaryName, versionStampPath } = require("./paths");

const REPO = process.env.KIROPROXY_REPO || "vtruong2k3/Kiro-Go";

// Maps the running platform to the asset name produced by the release workflow
// (kiro-go-{GOOS}-{GOARCH}). Anything absent here is unsupported.
const ASSETS = {
  "linux-x64": "kiro-go-linux-amd64",
  "linux-arm64": "kiro-go-linux-arm64",
  "darwin-x64": "kiro-go-darwin-amd64",
  "darwin-arm64": "kiro-go-darwin-arm64",
  "win32-x64": "kiro-go-windows-amd64.exe",
  "win32-arm64": "kiro-go-windows-arm64.exe",
};

function assetName() {
  const key = `${process.platform}-${process.arch}`;
  const name = ASSETS[key];
  if (!name) {
    throw new Error(
      `unsupported platform ${key}. Supported: ${Object.keys(ASSETS).join(", ")}. ` +
        `Build from source: https://github.com/${REPO}`
    );
  }
  return name;
}

function releaseBase(version) {
  const override = process.env.KIROPROXY_RELEASE_BASE;
  if (override) return override.replace(/\/+$/, "");
  return `https://github.com/${REPO}/releases/download/v${version}`;
}

// Minimal HTTPS GET that follows redirects — GitHub release assets always
// redirect to objects.githubusercontent.com, so this is not optional.
function httpsGet(url, { redirects = 5 } = {}) {
  return new Promise((resolve, reject) => {
    const target = new URL(url);
    if (target.protocol !== "https:") {
      reject(new Error(`refusing non-https download: ${url}`));
      return;
    }
    const req = https.get(
      target,
      { headers: { "User-Agent": "kiroproxy-cli" } },
      (res) => {
        const { statusCode, headers } = res;
        if (statusCode >= 300 && statusCode < 400 && headers.location) {
          res.resume();
          if (redirects <= 0) {
            reject(new Error(`too many redirects for ${url}`));
            return;
          }
          resolve(
            httpsGet(new URL(headers.location, target).toString(), {
              redirects: redirects - 1,
            })
          );
          return;
        }
        if (statusCode !== 200) {
          res.resume();
          reject(new Error(`HTTP ${statusCode} for ${url}`));
          return;
        }
        resolve(res);
      }
    );
    req.on("error", reject);
    req.setTimeout(60_000, () => {
      req.destroy(new Error(`timed out downloading ${url}`));
    });
  });
}

async function fetchBuffer(url) {
  const res = await httpsGet(url);
  const chunks = [];
  for await (const chunk of res) chunks.push(chunk);
  return Buffer.concat(chunks);
}

// Parses `sha256sum` output: "<hex>  <filename>" per line.
function parseSums(text) {
  const sums = new Map();
  for (const line of text.split("\n")) {
    const m = line.trim().match(/^([0-9a-f]{64})\s+\*?(.+)$/i);
    if (m) sums.set(path.basename(m[2]), m[1].toLowerCase());
  }
  return sums;
}

/**
 * Download the server binary for `version` into ~/.kiroproxy/bin.
 * Throws on any failure; callers decide whether that is fatal.
 */
async function downloadBinary(version, { log = () => {} } = {}) {
  const asset = assetName();
  const base = releaseBase(version);

  log(`downloading ${asset} (v${version})`);
  const [binary, sumsText] = await Promise.all([
    fetchBuffer(`${base}/${asset}`),
    fetchBuffer(`${base}/SHA256SUMS`),
  ]);

  const expected = parseSums(sumsText.toString("utf8")).get(asset);
  if (!expected) {
    throw new Error(`SHA256SUMS has no entry for ${asset}`);
  }
  const actual = crypto.createHash("sha256").update(binary).digest("hex");
  if (actual !== expected) {
    throw new Error(
      `checksum mismatch for ${asset}\n  expected ${expected}\n  got      ${actual}`
    );
  }

  fs.mkdirSync(binDir(), { recursive: true });
  // Write to a temp name and rename: a crash mid-write must not leave a
  // truncated binary that later looks installed.
  const finalPath = binaryPath();
  const tmpPath = path.join(binDir(), `.${binaryName()}.tmp`);
  fs.writeFileSync(tmpPath, binary, { mode: 0o755 });
  fs.renameSync(tmpPath, finalPath);
  fs.chmodSync(finalPath, 0o755);
  fs.writeFileSync(versionStampPath(), version, "utf8");

  log(`installed ${finalPath}`);
  return finalPath;
}

/** Version recorded for the currently installed binary, or null. */
function installedVersion() {
  try {
    return fs.readFileSync(versionStampPath(), "utf8").trim() || null;
  } catch {
    return null;
  }
}

/** True when a usable binary for `version` is already on disk. */
function isBinaryCurrent(version) {
  try {
    fs.accessSync(binaryPath(), fs.constants.X_OK);
  } catch {
    return false;
  }
  return installedVersion() === version;
}

module.exports = {
  ASSETS,
  assetName,
  downloadBinary,
  installedVersion,
  isBinaryCurrent,
  parseSums,
};
