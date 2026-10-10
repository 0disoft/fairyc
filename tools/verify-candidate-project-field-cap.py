"""Record the shared project layout/cleanup field cap and final fixture result."""
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
    path = CACHE / "native-project-field-cap.json"
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (2, 2, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    capacity = "tests::module_project_fields_capacity::project_layout_and_cleanup_fields_above_one_thousand_link_and_revalidate"
    cleanup = "tests::module_project_link::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    assert set(cases) == {capacity, cleanup}
    final_path = CACHE / "native-project-field-cap-final.json"
    final = json.loads(final_path.read_bytes())
    assert final["engine"] == "native" and final["profile"] == "development"
    assert (final["count"], final["passed"], final["failed"]) == (1, 1, 0) and final["tests"][0]["identity"] == capacity
    cases[capacity] = final["tests"][0]
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    assert base64.b64decode(cases[capacity]["stdoutBase64"], validate=True).decode() == "layout fields 1105 cleanup fields 1105\n"
    before = CACHE / "run-main-link-next-diagnostic.json"
    diagnostic = json.loads(before.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "compiled units 51 retries 28\nlink code 409102 module 98 at 0..0\n"
    generated_hash = hashlib.sha256(base64.b64decode(cases[cleanup]["stdoutBase64"], validate=True)).hexdigest()
    execution_path = CACHE / "project-link-c/execution.json"
    execution = json.loads(execution_path.read_bytes())
    assert execution["passed"]
    runs = [item for item in execution["executions"] if item["name"] == cleanup.split("::")[-1]]
    assert len(runs) == 2 and {item["profile"] for item in runs} == {"low", "optimized"}
    for item in runs:
        assert item["sourceSHA256"] == generated_hash and item["build"]["exit"] == item["run"]["exit"] == 0
        assert item["run"]["stdout"] == "7," and not item["run"]["stderr"]
    check = CACHE / "check-project-field-cap.json"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    attempt_path = CACHE / "run-main-link-field-cap-probe.json"
    attempt = json.loads(attempt_path.read_bytes())
    assert attempt["exit"] == 2 and attempt["stdout"] == "compiled units 51 retries 28\nlink code 1002 module 113 at 0..0\n" and not attempt["stderr"]
    project = CACHE / "main-link-field-cap-probe"
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert (project / "src" / source.name).read_bytes() == source.read_bytes(), source
    evidence = dict(schema="fairyc.candidate-project-field-cap/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("module_project_metadata.fy", "module_project_cleanup.fy", "module_unit_cleanup_mir_verify.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_fields_capacity.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        initialFailure=dict(report=identity(before), temporaryCode=409102, meaning="Metadata layout/field-cap guard 9102 plus linker metadata-stage tag 400000"),
        policy=dict(projectMetadataAndCleanupFields=2048, sourceAndJoinedASTFields=1024, layoutCap=1024, callableCap=1024, work=1048576),
        native=dict(initial=identity(path), finalCapacity=identity(final_path), passedCases=2, fields=1105, cleanupFields=1105,
            fixtureFinalization="Generated resource visibility was made explicit; the finalized capacity fixture was selectively rerun with unchanged source bytes. Unchanged cleanup case is reused.",
            coverage=["Two source owners keep their separate field arenas below 1024", "1105 aggregate layout and cleanup fields link and owned IR independently revalidates"]),
        reusedCExecution=dict(report=identity(execution_path), sourceSHA256=generated_hash, runs=2,
            justification="Fresh cleanup C is byte-identical to previously executed low/optimized C"),
        check=identity(check), fullOriginalLinkPassed=False,
        originalAttempt=dict(report=identity(attempt_path), stdout=attempt["stdout"], seconds=attempt["seconds"],
            wrapper=identity(project / "src/main.fy"), image=identity(project / "build/main-link-field-cap-probe.exe"),
            originalCompilationPassed=True, units=51, retries=28, nextModule="examples/lexer_candidate/src/module_unit_drops.fy",
            interpretation="The metadata guard at module 98 is passed; a later capacity failure still needs phase attribution before further changes"),
        deferred=["Remaining original link capacities", "Original dispatcher C execution", "Default compilation timeout/performance", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-field-cap-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, fields=1105, cleanupFields=1105, reusedCExecutions=2)))


if __name__ == "__main__":
    main()
