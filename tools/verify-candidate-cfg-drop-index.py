"""Verify scoped cleanup lookup, checked CFG batches and exact original root CFG."""
import base64
import hashlib
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-cfg-drop-index-probe"
IMPLEMENTATION = ("module_unit_cfg_drops.fy", "module_unit_cfg_lower.fy", "module_unit_cfg_model.fy", "module_unit_cfg_region.fy", "module_unit_c_emit.fy")


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def read(name):
    return json.loads((CACHE / name).read_bytes())


def main():
    native = read("native-cfg-drop-index.json")
    region = read("native-cfg-drop-index-region-with-sdk.json")
    assert (native["count"], native["passed"], native["failed"]) == (4, 4, 0)
    assert (region["count"], region["passed"], region["failed"]) == (1, 1, 0)
    for report in (native, region):
        assert report["engine"] == "native" and report["profile"] == "release"
        for case in report["tests"]:
            assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    stress = next(c for c in native["tests"] if c["identity"].endswith("::cfg_scopes_drop_queries_to_one_body_in_a_large_project"))
    assert base64.b64decode(stress["stdoutBase64"], validate=True) == b"drops 1440 blocks 503 work 3447\n"
    generated = next(c for c in native["tests"] if c["identity"].endswith("::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"))
    digest = hashlib.sha256(base64.b64decode(generated["stdoutBase64"], validate=True)).hexdigest()
    old = read("project-link-c/execution.json")
    reused = [r for r in old["executions"] if r["name"] == "project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"]
    assert old["passed"] and len(reused) == 2
    for run in reused:
        assert run["sourceSHA256"] == digest and run["build"]["exit"] == run["run"]["exit"] == 0
    check = read("check-cfg-drop-index-final.json")
    assert check["exit"] == 0 and "lock verified" in check["stdout"] and not check["stderr"]
    before = read("run-main-cfg-drop-diagnostic-probe.json")
    guard = read("cfg-drop-diagnostic-codes.json")["617005"]
    assert before["exit"] == 2 and "emit code 617005 module 0 at 1425..1435 C bytes 0" in before["stdout"]
    assert "while(i<clean_ir::drop_count(cleaned))" in guard
    for name in ("check", "build"):
        assert read(name + "-main-cfg-drop-index-probe.json")["exit"] == 0
    original = read("run-main-cfg-drop-index-closed-inputs-probe.json")
    expected = "compiled units 48 retries 25\nlinked frames 3558 work 3321980\nproject drops 6392 root drops 4\nroot CFG blocks 247 work 7650\nemit code 1002 module 0 at 0..0 C bytes 0\n"
    assert original["exit"] == 2 and not original["timedOut"] and original["stdout"] == expected and not original["stderr"]
    phase = read("run-main-cfg-c-phase-diagnostic-probe.json")
    assert phase["exit"] == 2 and phase["stdout"] == expected.replace("emit code 1002", "emit code 721007") and not phase["stderr"]
    stage = read("cfg-c-phase-diagnostic-codes.json")["721007"]
    assert "literals::definitions" in stage
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert source.read_bytes() == (PROBE / "src" / source.name).read_bytes(), source
    paths = read("main-cfg-drop-index-closed-inputs.json")
    assert len(paths) == len(set(paths)) == 236
    modules = {re.search(r"^module\s+([^;]+);", (ROOT / path).read_text(), re.M).group(1): path for path in paths}
    seen, pending = set(), ["main"]
    while pending:
        module = pending.pop()
        if module in seen:
            continue
        assert module in modules
        seen.add(module)
        pending.extend(re.findall(r"^import\s+([^\s]+)\s+as", (ROOT / modules[module]).read_text(), re.M))
    assert len(seen) == len(paths)
    missing_sdk = read("native-cfg-drop-index-region.json")
    assert missing_sdk["failed"] == 1
    correct = read("native-cfg-drop-index-region-with-sdk-observation.json")
    assert correct["exit"] == 0 and "--" in correct["argv"] and correct["argv"][-1].endswith("alloc.fy")
    misplaced = read("run-main-cfg-drop-index-probe.json")
    assert misplaced["exit"] == 134 and misplaced["stdout"] == "signatures 3\n"
    evidence = dict(schema="fairyc.candidate-cfg-drop-index/v1", passed=True,
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in IMPLEMENTATION],
        tests=[identity(ROOT / "examples/lexer_candidate/tests" / name) for name in ("module_unit_cfg.fy", "module_unit_cfg_region.fy")],
        receiptTool=identity(Path(__file__)), contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        native=dict(report=identity(CACHE / "native-cfg-drop-index.json"), region=identity(CACHE / "native-cfg-drop-index-region-with-sdk.json"), passed=5,
            stress=dict(projectFrames=1453, globalDrops=1440, rootDrops=0, rootBlocks=503, producerWork=3447),
            coverage=["Private per-frame original drop IDs", "Batch/single CFG identity", "Missing cleanup and nine existing CFG mutations reject", "Nested Region return/propagation preserves defer/drop/reset ordering", "Shared cleanup C remains byte-identical"]),
        sourceCheck=identity(CACHE / "check-cfg-drop-index-final.json"),
        reusedC=dict(report=identity(CACHE / "project-link-c/execution.json"), runs=2, sourceSHA256=digest),
        allocator=identity(ROOT / "internal/sdkstdlib/sources/std/alloc.fy"),
        originalProject=dict(inputs=[identity(ROOT / path) for path in paths], exactRootImportClosure=True, count=236, units=48, retries=25, frames=3558,
            globalDrops=6392, rootDrops=4, rootCFGBlocks=247, rootCFGWork=7650, rootCFGPassed=True, cGenerationPassed=False,
            before=identity(CACHE / "run-main-cfg-drop-diagnostic-probe.json"), beforeMap=identity(CACHE / "cfg-drop-diagnostic-codes.json"), beforeCode=617005,
            after=identity(CACHE / "run-main-cfg-drop-index-closed-inputs-probe.json"), wrapper=identity(PROBE / "src/main.fy"), image=identity(PROBE / "build/main-cfg-drop-index-probe.exe"),
            nextFailure=dict(diagnostic=identity(CACHE / "run-main-cfg-c-phase-diagnostic-probe.json"), codeMap=identity(CACHE / "cfg-c-phase-diagnostic-codes.json"), temporaryCode=721007, phase="module_unit_c_literals::definitions", code=1002, transactionalOutputBytes=0)),
        invalidHarnessAttempts=[dict(report=identity(CACHE / "native-cfg-drop-index-region.json"), reason="Required allocator SDK source argument omitted; corrected unchanged fixture passes"),
            dict(report=identity(CACHE / "native-cfg-region-before-index-control.json"), reason="Before-index control also omitted that required argument; no product-regression claim is made"),
            dict(report=identity(CACHE / "run-main-cfg-drop-index-probe.json"), reason="Observer-only helper was added to target inputs without its unrelated imports; exact main closure is 236 and was rerun")],
        limits=dict(bodyWork=1048576, bodyBlocks=32768, aggregateCFGBlocksWork=1048576, capacityAndWorkBoundsUnchanged=True),
        deferred=["Complete original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-cfg-drop-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=5, sharedCReusedRuns=2, originalRootCFG=True, originalC=False, nextPhase="literal definitions")))


if __name__ == "__main__":
    main()
