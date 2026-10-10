"""Bind the separate project field limit to native boundary and cleanup results."""
import base64
import hashlib
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    path = CACHE / "native-project-field-boundary.json"
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (2, 2, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    boundary = "tests::module_project_field_boundary::project_fields_admit_4096_and_reject_4097"
    cleanup = "tests::module_project_link::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    assert set(cases) == {boundary, cleanup}
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    assert base64.b64decode(cases[boundary]["stdoutBase64"], validate=True).decode() == "project fields 4096 accepted; 4097 rejected; unit cap 2048\n"
    diagnostic_path = CACHE / "run-main-link-layout-diagnostic.json"
    diagnostic = json.loads(diagnostic_path.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "compiled units 51 retries 28\nlink code 410303 module 213 at 313..2054\n"
    check_path = CACHE / "check-project-field-boundary.json"
    check = json.loads(check_path.read_bytes())
    assert check["exit"] == 0 and "check ok:" in check["stdout"] and not check["stderr"]
    generated_hash = hashlib.sha256(base64.b64decode(cases[cleanup]["stdoutBase64"], validate=True)).hexdigest()
    execution_path = CACHE / "project-link-c/execution.json"
    execution = json.loads(execution_path.read_bytes())
    runs = [row for row in execution["executions"] if row["name"] == cleanup.split("::")[-1]]
    assert execution["passed"] and len(runs) == 2 and {row["profile"] for row in runs} == {"low", "optimized"}
    for row in runs:
        assert row["sourceSHA256"] == generated_hash and row["build"]["exit"] == row["run"]["exit"] == 0
    names = ("candidate_limits.fy", "module_project_metadata.fy", "module_project_cleanup.fy", "module_unit_body_ir_verify.fy", "module_unit_cleanup_mir_verify.fy")
    evidence = dict(schema="fairyc.candidate-project-field-boundary/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in names],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_field_boundary.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        originalFailure=dict(report=identity(diagnostic_path), temporaryCode=410303, moduleIndex=213,
            source="internal/sdkstdlib/sources/optional/std/vec.fy", admittedLayouts=313, requestedFields=2054,
            meaning="The 2048 project field cap failed, independently of the 1024 layout cap or work guards"),
        policy=dict(projectFields=4096, importedUnitFields=2048, sourceFields=1024, projectLayouts=1024,
            producerWork=1048576, depth=64),
        native=dict(report=identity(path), passedCases=2, acceptedFields=4096, rejectedFields=4097,
            positiveUnits=8, fieldsPerUnit=512, overflowUnit=8,
            coverage=["Project layout and cleanup field tables agree at the exact cap", "Owned IR revalidation accepts the merged table", "One additional independently checked field rejects with 1002", "Shared cleanup generation remains identical"],
            fixtureCorrection="Explicit usize counts avoid default i32 literal inference in the generated source fixture"),
        sourceCheck=identity(check_path),
        reusedCExecution=dict(report=identity(execution_path), sourceSHA256=generated_hash, runs=2,
            justification="Fresh shared-cleanup C is byte-identical to the previously executed low/optimized artifacts"),
        originalFullLinkPassed=False,
        deferred=["Complete original linking and C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-field-boundary-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, admittedFields=4096, rejectedFields=4097, reusedCExecutions=2)))


if __name__ == "__main__":
    main()
