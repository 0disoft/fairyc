"""Record bounded indexed project contracts and actual cross-unit call validation."""
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
    path = CACHE / "native-project-call-index.json"
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (3, 3, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    capacity = "tests::module_project_frames_capacity::ordinary_project_frames_above_one_thousand_keep_distinct_coordinates"
    generic = "tests::module_project_link::project_c_namespaces_generic_instances_from_distinct_units"
    duplicate = "tests::module_project_link::project_link_rejects_missing_and_duplicate_ordinary_bodies"
    assert set(cases) == {capacity, generic, duplicate}
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    assert base64.b64decode(cases[capacity]["stdoutBase64"], validate=True).decode() == "frames 1089 work 9571 callables 1080\n"
    before = CACHE / "run-link-field-diagnostic.json"
    diagnostic = json.loads(before.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "compiled units 51 retries 28\nlink code 409203 module 113 at 1024..2895\n"
    generated_hash = hashlib.sha256(base64.b64decode(cases[generic]["stdoutBase64"], validate=True)).hexdigest()
    execution_path = CACHE / "project-link-c/execution.json"
    execution = json.loads(execution_path.read_bytes())
    runs = [row for row in execution["executions"] if row["name"] == generic.split("::")[-1]]
    assert execution["passed"] and len(runs) == 2 and {row["profile"] for row in runs} == {"low", "optimized"}
    for row in runs:
        assert row["sourceSHA256"] == generated_hash and row["build"]["exit"] == row["run"]["exit"] == 0
        assert not row["run"]["stdout"] and not row["run"]["stderr"]
    check = CACHE / "check-project-call-index.json"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-project-call-index/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("candidate_limits.fy", "module_project_metadata.fy", "module_project_calls.fy", "module_unit_body_ir_verify.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_frames_capacity.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        initialFailure=dict(report=identity(before), temporaryCode=409203, callableCount=1024, parameters=2895,
            meaning="The original project reached the callable table cap, independently distinguished from field and work guards"),
        policy=dict(projectCallables=4096, perUnitCallables=512, indexSlots=8192, work=1048576,
            keys=["contract: module/declaration", "ordinary target: module/declaration", "specialized target: module/declaration/context"],
            duplicateKeysRejected=True, argumentResultEffectAndUnsafeChecksRetained=True),
        native=dict(report=identity(path), passedCases=3, frames=1089, callables=1080, calls=1080, frameWork=9571,
            fixtureCorrections=["Provider-only functions produced no imported callable contracts; added nine caller units referencing all 1080 providers",
                "The resulting full-link case exposed the old repeated contract/frame scans exceeding the work budget; both lookups are now indexed"],
            coverage=["18 independently compiled modules preserve every original and rebound coordinate", "Full link and owned IR revalidation with 1080 callable contracts and calls",
                "Distinct generic unit contexts retain their targets", "Missing and duplicate ordinary bodies reject"]),
        reusedCExecution=dict(report=identity(execution_path), sourceSHA256=generated_hash, runs=2,
            justification="Freshly generated generic namespace C is byte-identical to previously executed low/optimized C"),
        check=identity(check), fullOriginalLinkPassed=False,
        deferred=["Remaining original linking/C generation", "Original dispatcher execution", "Default compilation timeout/performance", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-call-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, frames=1089, callables=1080, calls=1080, reusedCExecutions=2)))


if __name__ == "__main__":
    main()
