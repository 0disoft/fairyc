"""Record project frame capacity, coordinate indexing and unchanged cleanup C."""
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
    path = CACHE / "native-project-frame-index.json"
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (3, 3, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    capacity = "tests::module_project_frames_capacity::ordinary_project_frames_above_one_thousand_keep_distinct_coordinates"
    cleanup = "tests::module_project_link::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    duplicate = "tests::module_project_link::project_link_rejects_missing_and_duplicate_ordinary_bodies"
    assert set(cases) == {capacity, cleanup, duplicate}
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    assert base64.b64decode(cases[capacity]["stdoutBase64"], validate=True).decode() == "frames 1080 work 9561\n"
    diagnostic_path = CACHE / "run-main-link-frame-diagnostic.json"
    diagnostic = json.loads(diagnostic_path.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "compiled units 51 retries 28\nlink code 309002 module 213 at 0..0\n"
    generated = base64.b64decode(cases[cleanup]["stdoutBase64"], validate=True)
    generated_hash = hashlib.sha256(generated).hexdigest()
    execution_path = CACHE / "project-link-c/execution.json"
    execution = json.loads(execution_path.read_bytes())
    assert execution["passed"]
    reused = [item for item in execution["executions"] if item["name"] == cleanup.split("::")[-1]]
    assert len(reused) == 2 and {item["profile"] for item in reused} == {"low", "optimized"}
    for item in reused:
        assert item["sourceSHA256"] == generated_hash
        assert item["build"]["exit"] == item["run"]["exit"] == 0 and item["run"]["stdout"] == "7," and not item["run"]["stderr"]
    check = CACHE / "check-project-frame-index.json"
    assert "check ok:" in check.read_text(encoding="utf-8-sig")
    evidence = dict(schema="fairyc.candidate-project-frame-index/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in ("module_project_frames.fy", "candidate_limits.fy", "module_unit_cleanup_mir_verify.fy")],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_frames_capacity.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        initialFailure=dict(report=identity(diagnostic_path), temporaryCode=309002,
            meaning="Cache-only frame-cap guard 9002 plus linker frame-stage tag 300000 distinguishes the 1024-frame admission cap from work exhaustion"),
        policy=dict(projectFrames=4096, ordinaryHashSlots=8192, cleanupReceiverSlots="canonical type count + 1", work=1048576,
            ordinaryDuplicatesRejected=True, cleanupAliasesReused=True, genericUnitNamespacesPreserved=True, perUnitAdmissionUnchanged=True),
        native=dict(report=identity(path), passedCases=3, ordinaryFrames=1080, frameWork=9561,
            coverage=["1080 functions in nine source modules preserve all global and original coordinates", "Missing and duplicate ordinary bodies reject", "Shared cleanup bodies preserve generated C"]),
        reusedCExecution=dict(report=identity(execution_path), sourceSHA256=generated_hash, profiles=["low", "optimized"], runs=2,
            justification="Freshly generated cleanup C is byte-identical to the already executed C; existing successful profile executions remain applicable"),
        check=identity(check), fullOriginalLinkPassed=False,
        deferred=["Remaining original project link capacities", "Original dispatcher C execution", "Default compilation timeout/performance", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-frame-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, frames=1080, frameWork=9561, reusedCExecutions=2)))


if __name__ == "__main__":
    main()
