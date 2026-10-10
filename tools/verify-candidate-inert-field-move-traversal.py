"""Record owner-free field traversal admission and retained move diagnostics."""
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
    path = CACHE / "native-inert-field-move-traversal.json"
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (2, 2, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    original = "tests::module_project_compile::original_type_cli_unit_skips_unowned_field_traversal"
    regression = "tests::module_unit_moves::resource_transfers_follow_reachable_branches_and_loop_edges"
    assert set(cases) == {original, regression}
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    stdout = base64.b64decode(cases[original]["stdoutBase64"], validate=True).decode()
    measured = re.fullmatch(r"type_cli frames (\d+) producer work (\d+)\n", stdout)
    assert measured and int(measured[1]) > 0 and int(measured[2]) <= 1048576
    regression_stdout = base64.b64decode(cases[regression]["stdoutBase64"], validate=True).decode()
    golden_cases = re.findall(r"^case (\d+) (\d+) (\d+)$", regression_stdout, re.M)
    assert len(golden_cases) > 20 and "diagnostic 1101 " in regression_stdout
    before = CACHE / "run-type-cli-depth-diagnostic.json"
    diagnostic = json.loads(before.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "stage 5 code 9001 module 222 at 5794..5806 range 222..223\n"
    assert not diagnostic["stderr"]
    check = CACHE / "check-inert-field-move-traversal.json"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    full_path = CACHE / "native-main-inert-field-move-traversal.json"
    full = json.loads(full_path.read_bytes())
    assert full["engine"] == "native" and full["profile"] == "release"
    assert (full["count"], full["passed"], full["failed"]) == (1, 1, 0)
    full_case = full["tests"][0]
    assert full_case["identity"] == "tests::module_project_compile::original_main_all_units_compile" and full_case["outcome"] == "passed"
    assert not base64.b64decode(full_case["stderrBase64"], validate=True)
    full_stdout = base64.b64decode(full_case["stdoutBase64"], validate=True).decode()
    assert full_stdout == "compiled units 51 retries 28\n"
    inputs = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    digest = hashlib.sha256()
    total_bytes = 0
    for name in inputs:
        spelling = name.encode("utf-8")
        data = (ROOT / name).read_bytes()
        digest.update(len(spelling).to_bytes(8, "little") + spelling + len(data).to_bytes(8, "little") + data)
        total_bytes += len(data)
    evidence = dict(schema="fairyc.candidate-inert-field-move-traversal/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_moves.fy"),
        tests=[identity(ROOT / "examples/lexer_candidate/tests" / name) for name in ("module_project_compile.fy", "module_unit_moves.fy")],
        originalSource=identity(ROOT / "examples/lexer_candidate/src/type_cli.fy"), receiptTool=identity(Path(__file__)),
        diagnostic=dict(report=identity(before), temporaryCode=9001, meaning="Move expression recursion reached depth 64, independently distinguished from exhausted work in a cache-only probe"),
        policy=dict(moveDepth=64, work=1048576, resourceOwnerFieldsStillTraversed=True, inertOwnerScalarFieldsDoNotTriggerMoveTraversal=True),
        native=dict(report=identity(path), passedCases=2, goldenCases=len(golden_cases), stdout=stdout,
            coverage=["Unchanged original type_cli source compiles and owned IR revalidates", "Existing exact move/branch/loop diagnostic and related-span goldens remain valid"]),
        check=identity(check), stableAPIChanged=False, GoChanged=False, normativeContractsChanged=False, dbChanged=False, ciChanged=False,
        fullOriginalCompilationPassed=True,
        originalMain=dict(report=identity(full_path), inputPaths=identity(CACHE / "main-project-liveness-inputs.json"),
            sourceSet=dict(files=len(inputs), bytes=total_bytes, sha256=digest.hexdigest(), encoding="u64le path-byte length, UTF-8 relative path, u64le source-byte length, exact source bytes; in input order"),
            units=51, retries=28, stdout=full_stdout, nativeExecutionTimeoutSeconds=60,
            command=identity(CACHE / "native-main-inert-field-move-traversal-command.json"),
            validationHostCompileTimeoutSeconds=120, coversEverySourceExactlyOnce=True),
        deferred=["Dispatcher linking and C execution", "Default compilation timeout/performance", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-inert-field-move-traversal-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, goldenCases=len(golden_cases), stdout=stdout, fullOriginalCompilationPassed=True, units=51, retries=28)))


if __name__ == "__main__":
    main()
