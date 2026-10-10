"""Record native root-unit typing and external-call contract evidence."""
import base64
import json
import re
import runpy
import subprocess
from pathlib import Path
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[1]
source_set = SimpleNamespace(**runpy.run_path(ROOT / "tools/verify-candidate-main-source-set.py"))
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def native(name, expected, repaired=None):
    path = CACHE / name
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert report["target"] == "x86_64-windows-msvc"
    assert report["count"] == expected
    failures = 0
    for test in report["tests"]:
        if test["outcome"] != "passed":
            assert repaired and test["identity"] in repaired
            failures += 1
        else:
            assert not base64.b64decode(test["stderrBase64"], validate=True)
    assert failures == report["failed"] and report["passed"] == expected - failures
    return path, report


def main():
    root_path, root = native("native-main-unit-closure-final.json", 1)
    # A corrected syntax-only nested-call fixture may be rerun independently.
    nested_path = CACHE / "native-unit-nested-final.json"
    nested = None
    if nested_path.exists():
        nested_path, nested = native(nested_path.name, 1)
        assert nested["tests"][0]["identity"] == "tests::module_unit_external_calls::nested_calls_keep_authority_handling_and_unsafe_context"
    private_path, private = native("native-unit-private-final.json", 1)
    private_id = "tests::module_unit_calls::private_call_results_close_nested_fields_and_cleanup_call_results"
    assert private["tests"][0]["identity"] == private_id
    repaired = {private_id}
    if nested:
        repaired.add(nested["tests"][0]["identity"])
    calls_path, calls = native("native-unit-external-final.json", 8, repaired)
    debug_path = CACHE / "native-unit-private-debug.json"
    debug = json.loads(debug_path.read_bytes())
    assert debug["failed"] == debug["count"] == 1
    assert base64.b64decode(debug["tests"][0]["stdoutBase64"], validate=True).decode() == "1002 at 1 136..140\n"
    assert root["tests"][0]["identity"] == "tests::module_source_closure::original_main_call_closure_and_values"
    stdout = base64.b64decode(root["tests"][0]["stdoutBase64"], validate=True).decode()
    measured = re.fullmatch(r"root bound\nretained bodies (\d+) passes (\d+)\ntyped bodies (\d+) work (\d+)\n", stdout)
    assert measured and int(measured[1]) == 0 and int(measured[3]) == 4
    sources = source_set.closure()
    listed = json.loads((CACHE / "main-source-set-list.json").read_bytes())
    assert listed == [path.relative_to(ROOT).as_posix() for path in sources]
    check_path = CACHE / "check-main-unit-closure.txt"
    locked_path = CACHE / "check-main-unit-closure-locked.txt"
    checked = re.fullmatch(r"check ok: 2 instance\(s\), (\d+) source file\(s\), (\d+) MIR function\(s\), lock verified\n", locked_path.read_text(encoding="utf-8-sig"))
    assert checked and "check ok:" in check_path.read_text(encoding="utf-8-sig")
    changed = [ROOT / "examples/lexer_candidate/src" / name for name in (
        "module_unit_interfaces.fy", "module_unit_call_closure.fy", "module_unit_calls.fy", "module_unit_call_contracts.fy")]
    evidence = dict(schema="fairyc.candidate-main-unit-closure/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        sourceSet=dict(files=len(sources), rawBytes=sum(path.stat().st_size for path in sources),
                       method="Exact main.fy transitive import paths passed to native root fixture"),
        sources=[source_set.identity(path) for path in sources],
        changedSources=[source_set.identity(path) for path in changed],
        tests=[source_set.identity(ROOT / "examples/lexer_candidate/tests" / name) for name in (
            "module_source_closure.fy", "module_unit_external_calls.fy", "module_unit_calls.fy")],
        receiptTool=source_set.identity(Path(__file__)),
        rootUnit=dict(report=source_set.identity(root_path), stdout=stdout, retainedExternalBodies=int(measured[1]),
                      closurePasses=int(measured[2]), typedContexts=int(measured[3]), work=int(measured[4])),
        externalCalls=dict(report=source_set.identity(calls_path), passed=8, initialPassed=calls["passed"],
                           identities=[test["identity"] for test in calls["tests"]],
                           coverage=["Caller and provider type-check independently; signatures retain source coordinates",
                                     "Unlinked ordinary call lowers but standalone C rendering rejects absent peer body",
                                     "Legacy transitive closure still owns provider body",
                                     "Bad argument type, absent effect authority and ignored must-use reject",
                                     "Unsafe calls require lexical block or unsafe function authority",
                                     "Nested loop/if calls preserve handling and effect checks"],
                           nestedRetry=source_set.identity(nested_path) if nested else None,
                           privateRetry=source_set.identity(private_path),
                           fixtureDiagnosis=source_set.identity(debug_path),
                           initialFixtureCorrection="The nested-loop fixture now uses let ref T item syntax. A legacy fixture's struct-owned resource field correctly rejected with 1002; its outer type is now a resource with cleanup, retaining the original nested cleanup-call intent. Only these two cases are rerun; six unchanged passing cases are reused."),
        sourceCheck=dict(check=source_set.identity(check_path), locked=source_set.identity(locked_path),
                         sourceFiles=int(checked[1]), mirFunctions=int(checked[2])),
        bootstrap=dict(image=source_set.identity(ROOT / ".cache/candidate-tagged/compiler-file-tokens-120s.exe"),
                       hostCompileTimeoutSeconds=120, nativeExecutionTimeoutSeconds=60),
        deferred=["All dispatcher provider units and project body linking", "Whole-dispatcher C generation/execution",
                  "Native driver integration", "Default compilation timeout/performance", "Standalone candidate development/Release",
                  "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    output = ROOT / "docs/ops/evidence/candidate-main-unit-closure-20261011.json"
    output.write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, sources=len(sources), rootWork=int(measured[4]), nativePassed=9)))


if __name__ == "__main__":
    main()
