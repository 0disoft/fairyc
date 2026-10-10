"""Record field reborrows through reference call results and exact C executions."""
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


def report(name, profile, count):
    path = CACHE / name
    value = json.loads(path.read_bytes())
    assert value["engine"] == "native" and value["profile"] == profile and value["target"] == "x86_64-windows-msvc"
    assert (value["count"], value["passed"], value["failed"]) == (count, count, 0)
    for case in value["tests"]:
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    return value, identity(path)


def main():
    typed, typed_id = report("native-reference-call-fields.json", "development", 2)
    native_c, native_c_id = report("native-reference-call-fields-c.json", "development", 1)
    generated = base64.b64decode(native_c["tests"][0]["stdoutBase64"], validate=True)
    assert generated.startswith(b"/* fairyc.candidate-module-owned-c/v1 */\n")
    execution_path = CACHE / "reference-call-fields-c/execution.json"
    execution = json.loads(execution_path.read_bytes())
    assert execution["passed"] and execution["suite"] == "reference-call-fields" and len(execution["executions"]) == 2
    assert execution["nativeReport"]["sha256"] == native_c_id["sha256"]
    assert {x["profile"] for x in execution["executions"]} == {"low", "optimized"}
    for item in execution["executions"]:
        assert item["sourceSHA256"] == hashlib.sha256(generated).hexdigest()
        assert item["build"]["exit"] == item["run"]["exit"] == 0
        assert item["run"]["stdout"] == item["run"]["stderr"] == ""
    full_path = CACHE / "native-main-project-reference-fields.json"
    full = json.loads(full_path.read_bytes())
    assert full["engine"] == "native" and full["profile"] == "release" and full["count"] == 1
    case = full["tests"][0]
    stdout = base64.b64decode(case["stdoutBase64"], validate=True).decode()
    assert not base64.b64decode(case["stderrBase64"], validate=True)
    passed = case["outcome"] == "passed"
    source_paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    if passed:
        match = re.fullmatch(r"compiled units (\d+) retries (\d+)\n", stdout)
        assert match and full["passed"] == 1 and full["failed"] == 0
        observation = dict(compiledUnits=int(match[1]), retries=int(match[2]))
    else:
        match = re.fullmatch(r"stage (\d+) code (\d+) module (\d+) at (\d+)\.\.(\d+) range (\d+)\.\.(\d+)\n", stdout)
        assert match and full["passed"] == 0 and full["failed"] == 1
        stage, code, module, start, end, first, stop = map(int, match.groups())
        assert (stage, code, module, start, end) != (4, 2, 106, 12641, 12692)
        observation = dict(stage=stage, code=code, moduleIndex=module, start=start, end=end, first=first, stop=stop,
                           attributedSource=source_paths[module], unitFirstSource=source_paths[first])
    check = CACHE / "check-reference-call-fields.txt"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-reference-call-field-reborrows/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_value_place.fy"),
        tests=[identity(ROOT / "examples/lexer_candidate/tests" / name) for name in ("module_unit_places.fy", "module_unit_c_references.fy")],
        tools=[identity(Path(__file__)), identity(ROOT / "tools/verify-candidate-module-c.py")],
        native=dict(reports=[typed_id, native_c_id], passedCases=3),
        generatedC=dict(receipt=identity(execution_path), executions=2, bytes=len(generated), profiles=["/Od", "/O2"]),
        behavior=["Shared and exclusive reference call results denote existing referent storage", "Projected fields preserve reference mutability and resource restrictions",
                  "By-value temporary field borrows, shared-to-exclusive reborrows and resource-field exclusive borrows still reject",
                  "Unknown whole-owner origins remain zero; no synthetic local root or general interprocedural borrow proof is claimed"],
        check=identity(check), originalCompilationPassed=passed,
        original=dict(report=identity(full_path), stdout=stdout, files=len(source_paths), observation=observation,
                      sourceSet=[identity(ROOT / p) for p in source_paths]),
        deferred=["Remaining original compilation failures" if not passed else "Original dispatcher linking", "Original dispatcher C execution", "Linux and Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-reference-call-field-reborrows-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, generatedCRuns=2, originalCompilationPassed=passed, observation=observation)))


if __name__ == "__main__":
    main()
