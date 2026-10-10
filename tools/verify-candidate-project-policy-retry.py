"""Record the complete-main retry after recognizing raw cleanup-policy capacity errors."""
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
    before = ROOT / "docs/ops/evidence/candidate-cleanup-type-liveness-20261011.json"
    previous = json.loads(before.read_bytes())
    assert previous["original"]["nextFailure"] == "stage 6 code 10 module 44 at 0..0 range 44..50\n"
    for record in previous["original"]["sourceSet"]:
        assert identity(ROOT / record["path"])["sha256"] == record["sha256"]
    path = CACHE / "native-main-project-policy-retry.json"
    data = json.loads(path.read_bytes())
    assert data["engine"] == "native" and data["target"] == "x86_64-windows-msvc" and data["profile"] == "release" and data["count"] == 1
    case = data["tests"][0]
    assert case["identity"] == "tests::module_project_compile::original_main_all_units_compile"
    assert not base64.b64decode(case["stderrBase64"], validate=True)
    stdout = base64.b64decode(case["stdoutBase64"], validate=True).decode()
    passed = case["outcome"] == "passed"
    if passed:
        result = re.fullmatch(r"compiled units (\d+) retries (\d+)\n", stdout)
        assert result and data["passed"] == 1 and data["failed"] == 0
        observation = dict(compiledUnits=int(result[1]), retries=int(result[2]))
    else:
        result = re.fullmatch(r"stage (\d+) code (\d+) module (\d+) at (\d+)\.\.(\d+) range (\d+)\.\.(\d+)\n", stdout)
        assert result and data["passed"] == 0 and data["failed"] == 1
        stage, code, module, start, end, first, stop = map(int, result.groups())
        assert first >= 44 and (first, stop) != (44, 50)
        observation = dict(stage=stage, code=code, moduleIndex=module, start=start, end=end, first=first, stop=stop,
                           source=previous["original"]["sourceSet"][first]["path"])
    check = CACHE / "check-project-policy-retry.txt"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-project-cleanup-policy-retry/v1", retryValidated=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_project_compile.fy"), receiptTool=identity(Path(__file__)),
        before=identity(before), sourceSetReuse="All 236 original-file hashes match the previous receipt",
        native=identity(path), stdout=stdout, originalCompilationPassed=passed, observation=observation, check=identity(check),
        unchanged=["Non-capacity errors still propagate", "Single-source failures retain their stage and location", "All per-unit capacity limits"],
        deferred=["Remaining original compilation failures" if not passed else "Original dispatcher project linking", "Original dispatcher C execution", "Linux and Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-cleanup-policy-retry-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(retryValidated=True, originalCompilationPassed=passed, observation=observation)))


if __name__ == "__main__":
    main()
