import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";

const baseline = "2a5d2dd7dd05114154522fe22818ff299f60abda";
const current = "d5dc062b44e413f8e308bd402052513e034e7875";
const sha = (bytes: Buffer) => createHash("sha256").update(bytes).digest("hex");
const git = (...args: string[]) => execFileSync("git", args, { encoding: "utf8" }).trim();
if (git("rev-parse", "HEAD") !== current) throw Error("Review measurement revisions before reuse");
const root = path.resolve(".cache/argument-label-cost");
fs.mkdirSync(root, { recursive: true });
const replace: Record<string, string> = {};
const overlayHashes: Record<string, string> = {};
// Restore every changed production Go file, including exact SDK identity. Test
// sources stay current and the same new measurement harness serves both lanes.
for (const row of git("diff", "--name-status", baseline, current, "--", "internal").split(/\r?\n/)) {
  const [status, name] = row.split("\t");
  if (!name?.endsWith(".go") || name.endsWith("_test.go")) continue;
  if (status !== "A" && status !== "M") throw Error("Unexpected production change");
  const bytes = status === "A"
    ? Buffer.from(`package ${path.basename(path.dirname(name))}\n`)
    : execFileSync("git", ["show", `${baseline}:${name}`]);
  const saved = path.join(root, name.replaceAll("/", "_"));
  fs.writeFileSync(saved, bytes);
  replace[path.resolve(name)] = saved;
  overlayHashes[name] = sha(bytes);
}
const overlay = path.join(root, "overlay.json");
fs.writeFileSync(overlay, JSON.stringify({ Replace: replace }));
const baseEnv = { ...process.env, GOMAXPROCS: "1", FAIRYC_LABEL_COST_OUTPUT: root };
const binaries: Record<string, string> = {};
function run(command: string, args: string[], cwd: string, env = baseEnv, timeout = 120000) {
  return execFileSync(command, args, { cwd, env, encoding: "utf8", timeout, maxBuffer: 4 << 20 });
}
for (const lane of ["baseline", "current"]) {
  const binary = path.join(root, lane + ".test.exe");
  console.log("Build measurement binary: " + lane);
  run("bun", ["tools/run-go.ts", "test", "-mod=readonly", "-c", ...(lane === "baseline" ? ["-overlay", overlay] : []), "-o", binary, "./internal/cli"], process.cwd());
  binaries[lane] = binary;
}
const samples: any[] = [];
for (let repetition = 0; repetition < 6; repetition++) {
  const lanes = repetition % 2 ? ["current", "baseline"] : ["baseline", "current"];
  for (const lane of lanes) {
    console.log(`Measure ${repetition}: ${lane}`);
    const output = run(binaries[lane], ["-test.run=^TestArgumentLabelCostMeasurement$", "-test.v"], path.resolve("internal/cli"), { ...baseEnv, FAIRYC_LABEL_COST_LANE: lane, FAIRYC_LABEL_COST_REVERSE: String(repetition % 2) });
    const match = output.match(/LABEL_COST_REPORT (\{[^\r\n]+\})/);
    if (!match) throw Error("Missing measurement report");
    samples.push({ repetition, ...JSON.parse(match[1]) });
  }
}
console.log("Measure CodeView/PDB probes");
const debugOutput = run("bun", ["tools/run-go.ts", "test", "-mod=readonly", "-count=1", "-v", "./internal/cdriver", "-run", "^TestArgumentLabelDebugCost$"], process.cwd());
const debugMatch = debugOutput.match(/LABEL_DEBUG_REPORT (\[[^\r\n]+\])/);
if (!debugMatch) throw Error("Missing debug report");
const debug = JSON.parse(debugMatch[1]);
const cHashes = samples.flatMap(s => s.rows.filter((r: any) => r.variant !== "scanner").map((r: any) => r.c_sha256));
if (new Set(cHashes).size !== 1) throw Error("Generated C differs across lanes");
for (const variant of ["plain", "scanner"]) {
  const hashes = samples.flatMap(s => s.rows.filter((r: any) => r.variant === variant).map((r: any) => JSON.stringify(Object.fromEntries(Object.entries(r.source_sha256).filter(([p]) => p !== "fairy.toml").sort()))));
  if (new Set(hashes).size !== 1) throw Error("Unlabeled source corpus differs");
}
const harness: Record<string, string> = {};
for (const p of ["tools/measure-argument-labels.ts", "internal/cli/argument_labels_cost_test.go", "internal/cdriver/argument_labels_cost_test.go"]) harness[p] = sha(fs.readFileSync(p));
const report = { schema: "fairyc.argument-label-cost/v1", baseline, current, overlay_sha256: overlayHashes, harness_sha256: harness, binaries_sha256: Object.fromEntries(Object.entries(binaries).map(([k,v]) => [k,sha(fs.readFileSync(v))])),
  method: "Six alternating process pairs, GOMAXPROCS=1; one warmup then eight uncached load/parse/check/verified-MIR iterations per corpus and process. Forced GC before timing; normal GC within. Go cumulative allocations, not peak/RSS or child allocations. Setup, lock creation, process startup, Go compilation and cache serialization excluded. Same unlabeled .fy bytes across revisions; manifests use each compiler's exact language pin. Scanner+SDK control is frontend-only. Synthetic 64-call corpus additionally measures ordinary C/native builds in both profiles, artifact bytes, in-memory package-cache bytes with verified warm hits; separate existing private /Z7 /DEBUG:FULL probes capture PDB. Ambient OS/tool caches; source spans and random MSVC temporary paths may affect metadata. Sizes are observations, not filtered by equality. No runtime-speed, Linux, full application or disk-cache/fymap conclusion.", samples, debug };
fs.writeFileSync("docs/ops/evidence/argument-label-cost-20260922.json", JSON.stringify(report, null, 2) + "\n");
console.log("Saved argument-label-cost-20260922.json; all native executions and C/source identity checks passed");
