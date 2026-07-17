import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";

const args = process.argv.slice(2);
if (args.length === 0) {
  console.error("run-go requires at least one Go command argument");
  process.exit(2);
}

const cacheRoot = resolve(".cache");
const goCache = resolve(cacheRoot, "go-build");
const goModCache = resolve(cacheRoot, "go-mod");
const goPath = resolve(cacheRoot, "go-path");
const goTemp = resolve(cacheRoot, "go-tmp");

await Promise.all(
  [goCache, goModCache, goPath, goTemp].map((directory) =>
    mkdir(directory, { recursive: true }),
  ),
);

const child = Bun.spawn({
  cmd: ["go", ...args],
  cwd: process.cwd(),
  stdin: "ignore",
  stdout: "inherit",
  stderr: "inherit",
  env: {
    ...process.env,
    GOCACHE: goCache,
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
