"""Record original-body admission under the shared parser/IR expression cap."""
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
    path = CACHE / "native-owned-body-expression-cap.json"
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (2, 2, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    original = "tests::module_project_compile::original_origin_flow_unit_compile"
    guard = "tests::module_unit_body_ir::body_ir_rejects_malformed_owned_body_tables"
    assert set(cases) == {original, guard}
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    stdout = base64.b64decode(cases[original]["stdoutBase64"], validate=True).decode()
    measured = re.fullmatch(r"maximum nodes (\d+) owned frames (\d+) work (\d+)\n", stdout)
    assert measured
    nodes, frames, work = map(int, measured.groups())
    assert 4160 < nodes <= 16384 and frames > 0 and work <= 1048576
    diagnostic = CACHE / "native-origin-flow-diagnostic.json"
    before = json.loads(diagnostic.read_bytes())
    assert (before["passed"], before["failed"]) == (0, 1)
    assert base64.b64decode(before["tests"][0]["stdoutBase64"], validate=True).decode() == "stage 6 code 40004 module 138 at 0..0 range 138..139\n"
    check = CACHE / "check-owned-body-expression-cap.json"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    assert paths[138].endswith("/module_unit_origin_flow.fy")
    evidence = dict(schema="fairyc.candidate-owned-body-expression-cap/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_body_ir_verify.fy"),
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_compile.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        originalSource=identity(ROOT / paths[138]), sourceSet=[identity(ROOT / p) for p in paths],
        policy=dict(parserAndOwnedIRNodes=16384, work=1048576, otherLimitsUnchanged=True),
        diagnostic=dict(report=identity(diagnostic), temporaryCode=40004, meaning="check_body node/statement/text/region table-cap guard; temporary diagnostic numbering restored before final validation"),
        native=dict(report=identity(path), passedCases=2, maximumNodes=nodes, frames=frames, producerWork=work, stdout=stdout,
            coverage=["Original origin_flow unit admission and repeated owned-IR verification", "Malformed body tables and exhausted verifier work reject"]),
        check=identity(check), fullOriginalCompilationPassed=False,
        deferred=["Remaining original units", "Dispatcher linking and execution", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-owned-body-expression-cap-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, maximumNodes=nodes, frames=frames, producerWork=work)))


if __name__ == "__main__":
    main()
