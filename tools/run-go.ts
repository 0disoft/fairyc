import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";

const args = process.argv.slice(2);
if (args.length === 0) {
  console.error("run-go requires at least one Go command argument");
  process.exit(2);
}

const cacheRoot = resolve(".cache");
const goCache = resolve(cacheRoot, "go-build");
const goPath = resolve(cacheRoot, "go-path");
const goTemp = resolve(cacheRoot, "go-tmp");

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
const goModCache = new TextDecoder()
  .decode(moduleCacheProbe.stdout)
  .trim();
if (goModCache.length === 0) {
  console.error("run-go resolved an empty Go module cache path");
  process.exit(2);
}

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
    // Downloads stay disabled during validation. Reuse only exact go.sum-verified modules that a
    // separate dependency bootstrap has already placed in the user's standard module cache.
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
