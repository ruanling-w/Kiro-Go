#!/usr/bin/env node
"use strict";

// postinstall — warm the server binary into ~/.kiroproxy/bin so the first
// `kiroproxy` run starts instantly and works offline.
//
// Failure here is deliberately non-fatal: `npm i -g` must not fail because the
// user is behind a proxy or GitHub is briefly unreachable. cli.js retries the
// same download at startup, where an error can actually be reported in context.

const { downloadBinary, isBinaryCurrent } = require("../src/downloadBinary");
const pkg = require("../package.json");

async function main() {
  if (process.env.KIROPROXY_SKIP_DOWNLOAD === "1") {
    console.log("[kiroproxy] skipping binary download (KIROPROXY_SKIP_DOWNLOAD=1)");
    return;
  }
  if (isBinaryCurrent(pkg.version)) {
    console.log(`[kiroproxy] server binary v${pkg.version} already installed`);
    return;
  }
  await downloadBinary(pkg.version, {
    log: (m) => console.log(`[kiroproxy] ${m}`),
  });
  console.log("[kiroproxy] ready — run `kiroproxy` to start");
}

main().catch((err) => {
  console.warn(`[kiroproxy] binary download deferred: ${err.message}`);
  console.warn("[kiroproxy] it will be retried the first time you run `kiroproxy`");
  // Exit 0 on purpose — see the note at the top.
  process.exit(0);
});
