"""Record bounded unit coordination and the first complete-main compilation failure."""
import base64
import hashlib
import json
import re
import runpy
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def report(name, profile, passed, failed):
    path = CACHE / name
    value = json.loads(path.read_bytes())
    assert value["engine"] == "native" and value["profile"] == profile
    assert value["target"] == "x86_64-windows-msvc"
    assert value["count"] == passed + failed and value["passed"] == passed and value["failed"] == failed
    return value, identity(path)


def main():
    smoke, smoke_id = report("native-project-compile-smoke.json", "release", 2, 0)
    split, split_id = report("native-project-split.json", "development", 2, 0)
    for test in [*smoke["tests"], *split["tests"]]:
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
    original, original_id = report("native-main-project-compile-initial.json", "release", 0, 1)
    case = original["tests"][0]
    assert case["identity"] == "tests::module_project_compile::original_main_all_units_compile"
    stdout = base64.b64decode(case["stdoutBase64"], validate=True).decode()
    assert not base64.b64decode(case["stderrBase64"], validate=True)
    failure = re.fullmatch(r"stage (\d+) code (\d+) module (\d+) at (\d+)\.\.(\d+) range (\d+)\.\.(\d+)\n", stdout)
    assert failure and tuple(map(int, failure.groups())) == (6, 1002, 209, 10480, 44827, 5, 6)
    source_paths = runpy.run_path(str(ROOT / "tools/verify-candidate-main-source-set.py"))["closure"]()
    assert json.loads((CACHE / "main-project-inputs.json").read_bytes()) == [p.relative_to(ROOT).as_posix() for p in source_paths]
    check_path = CACHE / "check-project-compile-locked.txt"
    checked = re.fullmatch(r"check ok: 2 instance\(s\), (\d+) source file\(s\), (\d+) MIR function\(s\), lock verified\n", check_path.read_text(encoding="utf-8-sig"))
    assert checked
    evidence = dict(schema="fairyc.candidate-project-unit-coordination/v1", coordinatorValidated=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_project_compile.fy"),
        tests=[identity(ROOT / "examples/lexer_candidate/tests" / name) for name in ("module_project_compile.fy", "module_project_split.fy")],
        receiptTool=identity(Path(__file__)), native=[smoke_id, split_id], passedCases=4,
        coverage=["Grouped ordinary units cover every source and link", "Semantic errors retain stage, range and source span",
                  "80 generic specializations exceed the 64-per-unit cap; adaptive splitting produces three units with all 83 frames",
                  "Move diagnostics fail before returning owned IR"],
        sourceCheck=dict(receipt=identity(check_path), sourceFiles=int(checked[1]), mirFunctions=int(checked[2])),
        originalCompilationPassed=False, original=dict(report=original_id, sourceSet=[identity(p) for p in source_paths],
            files=len(source_paths), stdout=stdout, failedRange=[5, 6], failedSource=source_paths[5].relative_to(ROOT).as_posix(),
            attributedModule=source_paths[209].relative_to(ROOT).as_posix(), stage="owned IR", code=1002,
            interpretation="One-file bodies.fy unit reaches the SDK snapshot revalidation work cap; these offsets describe the retained SDK buffer, not the original fs.fy span"),
        limits=dict(sourceFiles=256, splitDepth=8, unitBudgetsUnchanged=True, completeResultOnly=True),
        bootstrap=dict(image=identity(ROOT / ".cache/candidate-tagged/compiler-file-tokens-120s.exe"), hostCompileTimeoutSeconds=120, nativeExecutionTimeoutSeconds=60),
        next=["Resolve measured owned-IR budget failure", "Compile remaining original units", "Link and execute original dispatcher C"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-unit-coordination-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(coordinatorValidated=True, passedCases=4, originalCompilationPassed=False, failedSource=evidence["original"]["failedSource"])))


if __name__ == "__main__":
    main()
