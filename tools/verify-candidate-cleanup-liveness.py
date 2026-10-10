"""Record cached cleanup type reachability on original compiler sources."""
import base64
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def report(name, profile, passed, failed):
    path = CACHE / name
    data = json.loads(path.read_bytes())
    assert data["engine"] == "native" and data["target"] == "x86_64-windows-msvc" and data["profile"] == profile
    assert (data["count"], data["passed"], data["failed"]) == (passed + failed, passed, failed)
    return data, identity(path)


def main():
    original, original_id = report("native-original-generic-expand-liveness.json", "development", 1, 0)
    regression, regression_id = report("native-cleanup-liveness-regression.json", "development", 1, 0)
    for case in [*original["tests"], *regression["tests"]]:
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    stdout = base64.b64decode(original["tests"][0]["stdoutBase64"], validate=True).decode()
    observed = re.fullmatch(r"generic_expand frames (\d+) producer work (\d+)\n", stdout)
    assert observed and int(observed[1]) == 100 and int(observed[2]) <= 1048576
    full, full_id = report("native-main-project-compile-liveness.json", "release", 0, 1)
    next_failure = base64.b64decode(full["tests"][0]["stdoutBase64"], validate=True).decode()
    assert next_failure == "stage 6 code 10 module 44 at 0..0 range 44..50\n"
    source_paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    check = CACHE / "check-cleanup-liveness.txt"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-cleanup-type-liveness/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("module_unit_cleanup_liveness.fy", "module_unit_cleanup_mir_lower.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_ir_budgets.fy"), receiptTool=identity(Path(__file__)),
        traversal=dict(typeCap=4096, depthCap=64, workCap=1048576, cacheEntriesPerType=1,
                       roots="Frame results, locals and resolved expressions", edges="Option/Result, arrays, references, typed pointers and iterator carriers",
                       unchanged="Nominal fields and Vec elements remain governed by their own cleanup recipes"),
        native=dict(reports=[original_id, regression_id], passedCases=2, originalFrames=int(observed[1]), producerWork=int(observed[2]),
                    coverage=["Complete original generic_expand.fy compiles under the unchanged producer cap", "Nested cleanup authority and malformed metadata rejection regression"]),
        check=identity(check), originalCompilationPassed=False,
        original=dict(report=full_id, sourceSet=[identity(ROOT / p) for p in source_paths], files=len(source_paths), nextFailure=next_failure,
                      nextRange=[44, 50], firstSource=source_paths[44], nextStep="Treat cleanup-policy code 10 as a lower-stage capacity failure and split this multi-source group"),
        deferred=["Remaining original units", "Original dispatcher linking and C execution"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-cleanup-type-liveness-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, originalFrames=int(observed[1]), producerWork=int(observed[2]), nextRange=[44, 50])))


if __name__ == "__main__":
    main()
