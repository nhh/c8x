#!/usr/bin/env node

const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");

const PLATFORMS = {
  "linux-x64": "@c8x/cli-linux-x64",
  "linux-arm64": "@c8x/cli-linux-arm64",
  "darwin-x64": "@c8x/cli-darwin-x64",
  "darwin-arm64": "@c8x/cli-darwin-arm64",
  "win32-x64": "@c8x/cli-win32-x64",
};

const platform = os.platform();
const arch = os.arch();
const key = `${platform}-${arch}`;
const pkg = PLATFORMS[key];

if (!pkg) {
  console.error(`Unsupported platform: ${platform}-${arch}`);
  console.error(`Supported: ${Object.keys(PLATFORMS).join(", ")}`);
  process.exit(1);
}

let binaryPath;
try {
  const pkgDir = path.dirname(require.resolve(`${pkg}/package.json`));
  const binaryName = platform === "win32" ? "c8x.exe" : "c8x";
  binaryPath = path.join(pkgDir, binaryName);
} catch {
  console.error(`Platform package ${pkg} is not installed.`);
  console.error(`Run: npm install ${pkg}`);
  process.exit(1);
}

try {
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (e) {
  if (e.status !== null) {
    process.exit(e.status);
  }
  throw e;
}
