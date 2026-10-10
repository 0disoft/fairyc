"""Bind project type-rebinding evidence to the current Fairy sources."""
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


def main():
    report_path = CACHE / "native-project-types.json"
    report = json.loads(report_path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert report["target"] == "x86_64-windows-msvc"
    assert report["count"] == report["passed"] == 3 and report["failed"] == 0
    expected = {"independently_lowered_units_share_project_types", "reordered_wrappers_rebind_edges_and_preserve_array_lengths",
                "project_type_and_unit_limits_and_signature_prefix_reject"}
    assert {test["identity"] for test in report["tests"]} == {"tests::module_project_types::" + name for name in expected}
    stdout = {}
    for test in report["tests"]:
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
        stdout[test["identity"]] = base64.b64decode(test["stdoutBase64"], validate=True).decode()
    measured = re.fullmatch(r"independent IR units 2 canonical types (\d+)\n", stdout["tests::module_project_types::independently_lowered_units_share_project_types"])
    assert measured and int(measured[1]) > 1
    checked_path = CACHE / "check-project-types.txt"
    locked_path = CACHE / "check-project-types-locked.txt"
    checked = re.fullmatch(r"check ok: 2 instance\(s\), (\d+) source file\(s\), (\d+) MIR function\(s\), lock verified\n", locked_path.read_text(encoding="utf-8-sig"))
    assert checked and "check ok:" in checked_path.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-project-type-rebinding/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        sources=[identity(ROOT / "examples/lexer_candidate/src/module_project_types.fy"),
                 identity(ROOT / "examples/lexer_candidate/tests/module_project_types.fy")], receiptTool=identity(Path(__file__)),
        native=dict(report=identity(report_path), passed=3, stdout=stdout, independentUnits=2, canonicalTypes=int(measured[1]),
                    coverage=["Independently bound, typed, moved and lowered caller/provider IR graphs share project canonical types",
                              "Differently numbered array, Option, Result and nominal pointer descriptors rebind to identical types",
                              "Array length and pointer mutability/ABI flags remain numeric payloads",
                              "256 units accepted, 257 rejected; 4096 project types accepted, the 4097th rejected",
                              "Mismatched signature prefixes and invalid map lookups fail closed"]),
        limits=dict(projectTypes=4096, sourceUnits=256, hashSlots=8192, work=1048576),
        sourceCheck=dict(check=identity(checked_path), locked=identity(locked_path), sourceFiles=int(checked[1]), mirFunctions=int(checked[2])),
        bootstrap=dict(image=identity(ROOT / ".cache/candidate-tagged/compiler-file-tokens-120s.exe"),
                       hostCompileTimeoutSeconds=120, nativeExecutionTimeoutSeconds=60),
        deferred=["Project function/cleanup body rebinding and linking", "Whole-dispatcher C generation/execution",
                  "Native driver integration", "Default compilation timeout/performance", "Standalone candidate development/Release",
                  "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-type-rebinding-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, independentUnits=2, canonicalTypes=int(measured[1]))))


if __name__ == "__main__":
    main()
