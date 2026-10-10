"""Record independently bounded ownership analysis in the project coordinator."""
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
    before_path = ROOT / "docs/ops/evidence/candidate-reference-call-field-reborrows-20261011.json"
    before = json.loads(before_path.read_bytes())
    assert before["original"]["stdout"] == "stage 5 code 10 module 114 at 0..0 range 114..115\n"
    for record in before["original"]["sourceSet"]:
        assert identity(ROOT / record["path"])["sha256"] == record["sha256"]
    regression_path = CACHE / "native-project-move-budget-regression.json"
    regression = json.loads(regression_path.read_bytes())
    assert regression["engine"] == "native" and regression["profile"] == "development"
    assert (regression["count"], regression["passed"], regression["failed"]) == (2, 2, 0)
    for case in regression["tests"]:
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    path = CACHE / "native-main-project-move-budget.json"
    data = json.loads(path.read_bytes())
    assert data["engine"] == "native" and data["target"] == "x86_64-windows-msvc" and data["profile"] == "release" and data["count"] == 1
    case = data["tests"][0]
    assert not base64.b64decode(case["stderrBase64"], validate=True)
    stdout = base64.b64decode(case["stdoutBase64"], validate=True).decode()
    passed = case["outcome"] == "passed"
    if passed:
        match = re.fullmatch(r"compiled units (\d+) retries (\d+)\n", stdout)
        assert match and data["passed"] == 1 and data["failed"] == 0
        observation = dict(compiledUnits=int(match[1]), retries=int(match[2]))
    else:
        match = re.fullmatch(r"stage (\d+) code (\d+) module (\d+) at (\d+)\.\.(\d+) range (\d+)\.\.(\d+)\n", stdout)
        assert match and data["passed"] == 0 and data["failed"] == 1
        stage, code, module, start, end, first, stop = map(int, match.groups())
        assert (stage, code, module, first, stop) != (5, 10, 114, 114, 115)
        observation = dict(stage=stage, code=code, moduleIndex=module, start=start, end=end, first=first, stop=stop,
                           attributedSource=before["original"]["sourceSet"][module]["path"])
    check = CACHE / "check-project-move-budget.txt"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-project-independent-move-budget/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_project_compile.fy"), receiptTool=identity(Path(__file__)),
        policy=dict(valuesWorkCap=1048576, ownershipWorkCap=1048576, ownershipInitialWork=0,
                    note="Project coordination bounds the completed value and ownership traversals independently; direct moves::build prior-work behavior is unchanged"),
        check=identity(check), nativeRegression=identity(regression_path), passedCases=2,
        coverage=["Type errors still propagate before lowering", "Move diagnostics still reject before returning owned IR"],
        before=identity(before_path), originalCompilationPassed=passed,
        original=dict(report=identity(path), stdout=stdout, observation=observation, files=len(before["original"]["sourceSet"]),
                      sourceSetReuse="All original source hashes match the previous receipt"),
        deferred=["Remaining original compilation failures" if not passed else "Original dispatcher linking", "Original dispatcher C execution", "Linux and Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-independent-move-budget-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, originalCompilationPassed=passed, observation=observation)))


if __name__ == "__main__":
    main()
