import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";

const sha256 = (bytes: Buffer | string) => createHash("sha256").update(bytes).digest("hex");
const root = path.resolve(".cache/argument-ast-layout");
fs.mkdirSync(root, { recursive: true });

function replaceOnce(source: string, before: string, after: string, file: string): string {
  const at = source.indexOf(before);
  if (at < 0 || source.indexOf(before, at + before.length) >= 0) {
    throw Error(`Expected one overlay boundary in ${file}: ${before.slice(0, 60)}`);
  }
  return source.slice(0, at) + after + source.slice(at + before.length);
}

const replacements: Record<string, string> = {};
const overlayHashes: Record<string, string> = {};
function overlay(file: string, change: (text: string) => string) {
  const original = fs.readFileSync(file, "utf8").replaceAll("\r\n", "\n");
  const altered = change(original);
  const target = path.join(root, file.replaceAll("/", "_"));
  fs.writeFileSync(target, altered);
  replacements[path.resolve(file)] = target;
  overlayHashes[file] = sha256(altered);
}

overlay("internal/ast/ast.go", (original) => {
  let text = replaceOnce(original, "\tCallLabels    *CallArgumentLabels `json:\",omitempty\"`\n", "", "ast.go");
  text = replaceOnce(text, `		if node.Kind == syntax.KindCall {
			if label, found := tree.Node(child); found && label.Kind == syntax.KindArgumentLabel && len(label.Children) == 1 {
				if expression.CallLabels == nil {
					expression.CallLabels = &CallArgumentLabels{}
				}
				expression.CallLabels.Labels = append(expression.CallLabels.Labels, ArgumentLabel{Index: len(expression.Children) - 1, Name: label.Text, Span: label.Span})
				child = label.Children[0]
			}
		}
`, "", "ast.go");
  text = replaceOnce(text, `	if expression.CallLabels != nil && len(expression.Children) > 0 {
		expression.CallLabels.ExplicitTypeArguments = expression.Children[0].Kind == syntax.KindSpecialize
	}
`, "", "ast.go");
  return text;
});
overlay("internal/genericsource/clone.go", (text) => replaceOnce(text, `	if v.CallLabels != nil {
		labels := *v.CallLabels
		labels.Labels = slices.Clone(labels.Labels)
		result.CallLabels = &labels
	}
`, "", "clone.go"));
overlay("internal/semantic/argument_labels.go", () => `package semantic

import "github.com/0disoft/fairyc/internal/ast"

func (a *analyzer) checkCallArgumentLabels(_ *moduleScope, _ *ast.Expression) bool { return true }
`);
overlay("internal/cli/fairy_parser_body_tree_test.go", (text) => {
  const start = "\t\t\tif k == 205 && e.CallLabels != nil {\n";
  const stop = "\t\t\tif k == 207 && i > 0 && c.Kind == syntax.KindBlock {\n";
  const at = text.indexOf(start), end = text.indexOf(stop, at + start.length);
  if (at < 0 || end < 0 || text.indexOf(start, at + start.length) >= 0) throw Error("Parser tree label boundary changed");
  return text.slice(0, at) + text.slice(end);
});
for (const file of ["internal/cli/argument_labels_ast_cost_test.go", "internal/cli/argument_labels_cache_attribution_test.go"]) {
  overlay(file, () => "package cli\n");
}
const overlayPath = path.join(root, "overlay.json");
fs.writeFileSync(overlayPath, JSON.stringify({ Replace: replacements }));

const environment = { ...process.env, GOMAXPROCS: "1" };
const run = (command: string, args: string[], cwd: string, env = environment, timeout = 120000) =>
  execFileSync(command, args, { cwd, env, encoding: "utf8", timeout, maxBuffer: 4 << 20 });

const binaries: Record<string, string> = {};
for (const lane of ["current", "without-label-field"]) {
  const binary = path.join(root, lane + ".test.exe");
  console.log(`Build AST layout lane: ${lane}`);
  run("bun", ["tools/run-go.ts", "test", "-mod=readonly", "-c", ...(lane === "current" ? [] : ["-overlay", overlayPath]), "-o", binary, "./internal/cli"], process.cwd(), environment, 180000);
  binaries[lane] = binary;
}

type Sample = {
  lane: string;
  go: string;
  gomaxprocs: number;
  iterations: number;
  source_sha256: Record<string, string>;
  ast_expression_bytes: number;
  mir_functions: number;
  mir_json_sha256: string;
  check_ns_per_iteration: number;
  go_alloc_bytes_per_iteration: number;
  go_mallocs_per_iteration: number;
  gc_cycles: number;
};
const pairs: Array<{ pair: number; first: string; current: Sample; compact: Sample }> = [];
for (let pair = 0; pair < 6; pair++) {
  const order = pair % 2 === 0 ? ["current", "without-label-field"] : ["without-label-field", "current"];
  const observations: Record<string, Sample> = {};
  for (const lane of order) {
    console.log(`Measure AST layout pair ${pair + 1}: ${lane}`);
    const output = run(binaries[lane], ["-test.run=^TestArgumentLabelsASTLayoutMeasure$", "-test.v"], path.resolve("internal/cli"), { ...environment, FAIRYC_AST_LAYOUT_LANE: lane });
    const match = output.match(/AST_LAYOUT_REPORT (\{[^\r\n]+\})/u);
    if (!match) throw Error(`No AST layout report for ${lane}: ${output}`);
    const sample = JSON.parse(match[1]) as Sample;
    if (sample.lane !== lane || sample.iterations !== 4 || sample.gomaxprocs !== 1) throw Error("Invalid sample identity");
    observations[lane] = sample;
  }
  const current = observations.current, compact = observations["without-label-field"];
  if (current.ast_expression_bytes !== 136 || compact.ast_expression_bytes !== 128) throw Error("Unexpected AST layout sizes");
  if (JSON.stringify(current.source_sha256) !== JSON.stringify(compact.source_sha256) ||
      current.mir_functions !== compact.mir_functions || current.mir_json_sha256 !== compact.mir_json_sha256) {
    throw Error("Compared lanes differ in source or verified MIR");
  }
  pairs.push({ pair: pair + 1, first: order[0], current, compact });
}

const median = (values: number[]) => {
  const sorted = [...values].sort((a, b) => a - b);
  return (sorted[2] + sorted[3]) / 2;
};
const timeRatios = pairs.map(({ current, compact }) => compact.check_ns_per_iteration / current.check_ns_per_iteration);
const allocDifferences = pairs.map(({ current, compact }) => compact.go_alloc_bytes_per_iteration - current.go_alloc_bytes_per_iteration);
const mallocDifferences = pairs.map(({ current, compact }) => compact.go_mallocs_per_iteration - current.go_mallocs_per_iteration);
const report = {
  schema: "fairyc.argument-ast-layout/v1",
  compiler_head: run("git", ["rev-parse", "HEAD"], process.cwd()).trim(),
  harness_sha256: {
    "tools/measure-argument-ast-layout.ts": sha256(fs.readFileSync("tools/measure-argument-ast-layout.ts")),
    "internal/cli/argument_labels_ast_layout_measure_test.go": sha256(fs.readFileSync("internal/cli/argument_labels_ast_layout_measure_test.go")),
  },
  overlay_sha256: overlayHashes,
  binary_sha256: Object.fromEntries(Object.entries(binaries).map(([lane, binary]) => [lane, sha256(fs.readFileSync(binary))])),
  method: "Six alternating isolated process pairs, GOMAXPROCS=1; four fresh load/parse/check/verified-MIR iterations after one warmup per process. Source setup, process startup, Go build, and MIR JSON hashing excluded. Go TotalAlloc and Mallocs are per-iteration cumulative differences, not retained heap or peak RSS. Experimental compact lane removes label support and test-only label inspections; both lanes process the same unlabeled scanner source. This is not a shippable compiler variant or a language-wide speed claim.",
  paired_median_time_ratio_compact_over_current: median(timeRatios),
  paired_median_alloc_byte_delta: median(allocDifferences),
  paired_median_malloc_delta: median(mallocDifferences),
  pairs,
};
const evidence = "docs/ops/evidence/argument-label-ast-layout-20260923.json";
fs.writeFileSync(evidence, JSON.stringify(report, null, 2) + "\n");
console.log(JSON.stringify({ evidence, paired_median_time_ratio_compact_over_current: report.paired_median_time_ratio_compact_over_current, paired_median_alloc_byte_delta: report.paired_median_alloc_byte_delta, paired_median_malloc_delta: report.paired_median_malloc_delta }));
