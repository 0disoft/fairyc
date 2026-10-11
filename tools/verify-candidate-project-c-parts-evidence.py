"""Bind private C partition execution to source and original-project observations."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    execution = json.loads((CACHE / "c-partition-native-c/execution.json").read_bytes())
    assert execution["passed"] and len(execution["cases"]) == 3
    assert all(len(case["executions"]) == 2 for case in execution["cases"].values())
    check = json.loads((CACHE / "check-c-partition-final.json").read_bytes())
    assert check["exit"] == 0 and "lock verified" in check["stdout"] and not check["stderr"]
    native = json.loads((CACHE / "native-c-partition-final.json").read_bytes())
    assert native["count"] == native["passed"] == 4 and native["failed"] == 0
    original = json.loads((CACHE / "run-main-c-partition-probe.json").read_bytes())
    assert original["exit"] == 2 and not original["timedOut"] and not original["stderr"]
    assert original["stdout"] == "compiled units 48 retries 25\nlinked frames 3558 work 3321980\nproject drops 6392 root drops 4\nroot CFG blocks 247 work 7650\npartition code 1004 module 209 at 30613..30660\n"
    paths = json.loads((CACHE / "main-cfg-drop-index-closed-inputs.json").read_bytes())
    assert len(set(paths)) == len(paths) == 236 and original["argv"][1:] == paths
    assert paths[209] == "internal/sdkstdlib/sources/std/fs.fy"
    assert b"read_dir_drop_raw(ref mut this as ptr mut void)" in (ROOT / paths[209]).read_bytes()[30613:30660]
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert source.read_bytes() == (CACHE / "main-c-partition-probe/src" / source.name).read_bytes(), source
    receipt = dict(schema="fairyc.candidate-project-c-parts/v1", passed=True,
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("module_unit_c_emit.fy", "module_unit_c_literals.fy")],
        tests=identity(ROOT / "examples/lexer_candidate/tests/module_project_link.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-project-c-parts-v0.md"),
        receiptTools=[identity(ROOT / "tools" / name) for name in ("verify-candidate-project-c-parts.py", "verify-candidate-project-c-parts-evidence.py")],
        sourceCheck=check, native=dict(report=identity(CACHE / "native-c-partition-final.json"), passed=4, observation=json.loads((CACHE / "native-c-partition-final-observation.json").read_bytes())),
        separateTranslationUnits=dict(report=identity(CACHE / "c-partition-native-c/execution.json"), cases=execution["cases"], executions=6),
        legacyLiteralDigest=execution["literalLegacyStress"],
        originalProject=dict(passed=False, sourceInputs=[identity(ROOT / path) for path in paths], observation=original, wrapper=identity(CACHE / "main-c-partition-probe/src/main.fy"), image=identity(CACHE / "main-c-partition-probe/build/main-c-partition-probe.exe"), blocker="Whole-frame emission includes unreachable SDK ReadDir cleanup without admitted C runtime ABI; entry reachability remains required"),
        limits=dict(unitWork=1048576, outputBytesPerBuffer=2097152, projectWork=4194304, splitDepth=12),
        deferred=["Entry-reachable function/cleanup selection", "Original complete C generation and execution", "Native driver/publication", "Default timeout/performance", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-c-parts-20261011.json").write_text(json.dumps(receipt, indent=2) + "\n")
    print("PASS: 4 native cases, 6 separate-TU executions; original ReadDir blocker retained")


if __name__ == "__main__":
    main()
