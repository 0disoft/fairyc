import { existsSync, readFileSync } from "node:fs";
import { mkdir } from "node:fs/promises";
import { resolve, sep } from "node:path";

const args = process.argv.slice(2);
if (args.length === 0) {
  console.error("run-go requires at least one Go command argument");
  process.exit(2);
}

const cacheRoot = resolve(".cache");
const goCache = resolve(cacheRoot, "go-build");
const goPath = resolve(cacheRoot, "go-path");
const goTemp = resolve(cacheRoot, "go-tmp");
const localModuleCache = resolve(cacheRoot, "go-mod");

await Promise.all(
  [goCache, goPath, goTemp].map((directory) =>
    mkdir(directory, { recursive: true }),
  ),
);

const moduleCacheProbe = Bun.spawnSync({
  cmd: ["go", "env", "GOMODCACHE"],
  cwd: process.cwd(),
  stdin: "ignore",
  stdout: "pipe",
  stderr: "pipe",
  env: {
    ...process.env,
    GOENV: "off",
    GOTOOLCHAIN: "local",
  },
});
if (moduleCacheProbe.exitCode !== 0) {
  const detail = new TextDecoder().decode(moduleCacheProbe.stderr).trim();
  console.error(`run-go could not resolve the shared Go module cache: ${detail}`);
  process.exit(2);
}
const sharedModuleCache = new TextDecoder()
  .decode(moduleCacheProbe.stdout)
  .trim();
if (sharedModuleCache.length === 0) {
  console.error("run-go resolved an empty Go module cache path");
  process.exit(2);
}

// Prefer an already bootstrapped local cache only when every module archive
// recorded in go.sum has the exact expected hash. An incomplete local cache
// must not hide a usable shared cache or trigger a network download.
const downloadRoot = resolve(localModuleCache, "cache", "download");
const checksums = readFileSync("go.sum", "utf8")
  .split(/\r?\n/u)
  .map((line) => line.trim().split(/\s+/u))
  .filter((parts) => parts.length === 3 && parts[2].startsWith("h1:") && !parts[1].endsWith("/go.mod"));
const localReady = checksums.length > 0 && checksums.every(([module, version, expected]) => {
  const escaped = module.replace(/[A-Z]/gu, (letter) => `!${letter.toLowerCase()}`);
  const zipHash = resolve(downloadRoot, escaped, "@v", `${version}.ziphash`);
  const zip = resolve(downloadRoot, escaped, "@v", `${version}.zip`);
  const metadata = resolve(downloadRoot, escaped, "@v", `${version}.mod`);
  const information = resolve(downloadRoot, escaped, "@v", `${version}.info`);
  return zipHash.startsWith(downloadRoot + sep) && existsSync(zipHash) && existsSync(zip) &&
    existsSync(metadata) && existsSync(information) &&
    readFileSync(zipHash, "utf8").trim() === expected;
});
const goModCache = localReady ? localModuleCache : sharedModuleCache;

const child = Bun.spawn({
  cmd: ["go", ...args],
  cwd: process.cwd(),
  stdin: "ignore",
  stdout: "inherit",
  stderr: "inherit",
  env: {
    ...process.env,
    // Project-owned C files are compiler inputs loaded as data. Disabling cgo
    // keeps Go from treating them as package-native sources on hosts with a C compiler.
    CGO_ENABLED: "0",
    GOCACHE: goCache,
    // Downloads stay disabled during validation. Reuse the exact go.sum-verified local cache
    // when complete, otherwise retain the user's standard shared module cache.
    GOMODCACHE: goModCache,
    GOPATH: goPath,
    GOTMPDIR: goTemp,
    GOENV: "off",
    GOTOOLCHAIN: "local",
    GOPROXY: "off",
    GOSUMDB: "off",
    TEMP: goTemp,
    TMP: goTemp,
  },
});

process.exit(await child.exited);
