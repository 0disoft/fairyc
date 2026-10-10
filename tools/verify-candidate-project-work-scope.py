"""Record explicit project work admission, bounded revalidation and main linking."""
import base64
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-link-work-scope-probe"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    report_path = CACHE / "native-project-work-scope.json"
    report = json.loads(report_path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (3, 3, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    cleanup = "tests::module_project_link::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    expected = {cleanup, "tests::module_unit_cleanup_mir::cleanup_mir_rejects_invalid_handoffs_and_missing_authority", "tests::module_unit_body_ir::body_ir_rejects_malformed_owned_body_tables"}
    assert set(cases) == expected
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    generated_hash = hashlib.sha256(base64.b64decode(cases[cleanup]["stdoutBase64"], validate=True)).hexdigest()
    execution_path = CACHE / "project-link-c/execution.json"
    executions = json.loads(execution_path.read_bytes())
    runs = [row for row in executions["executions"] if row["name"] == cleanup.split("::")[-1]]
    assert executions["passed"] and len(runs) == 2 and {row["profile"] for row in runs} == {"low", "optimized"}
    for row in runs:
        assert row["sourceSHA256"] == generated_hash and row["build"]["exit"] == row["run"]["exit"] == 0
    check_path = CACHE / "check-project-work-scope-final.json"
    check = json.loads(check_path.read_bytes())
    assert check["exit"] == 0 and "check ok:" in check["stdout"] and not check["stderr"]
    build_path = CACHE / "build-main-link-work-scope-probe.json"
    built = json.loads(build_path.read_bytes())
    probe_check_path = CACHE / "check-main-link-work-scope-probe.json"
    assert built["exit"] == json.loads(probe_check_path.read_bytes())["exit"] == 0 and not built["stderr"]
    run_path = CACHE / "run-main-link-work-scope-probe.json"
    run = json.loads(run_path.read_bytes())
    assert run["exit"] in (0, 2) and not run["stderr"]
    compiled = re.search(r"compiled units (\d+) retries (\d+)\n", run["stdout"])
    assert compiled
    linked = re.search(r"linked frames (\d+) work (\d+)\n", run["stdout"])
    assert (linked is not None) == (run["exit"] == 0)
    diagnostic_path = CACHE / "run-main-work-scope-diagnostic.json"
    diagnostic = json.loads(diagnostic_path.read_bytes())
    assert diagnostic["exit"] == 2 and diagnostic["stdout"] == "compiled units 48 retries 25\nlink code 510524 module 0 at 0..0\n"
    copied = []
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert (PROBE / "src" / source.name).read_bytes() == source.read_bytes(), source
            copied.append(identity(source))
    paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    inputs = [identity(ROOT / path) for path in paths]
    names = ["candidate_limits.fy", "module_project_body.fy", "module_project_body_equal.fy", "module_project_cleanup.fy", "module_project_calls.fy", "module_project_link.fy", "module_unit_cleanup_mir_model.fy", "module_unit_cleanup_mir_verify.fy", "module_unit_body_ir_verify.fy"]
    before_path = CACHE / "run-main-link-layout-index-probe.json"
    before = json.loads(before_path.read_bytes())
    assert before["exit"] == 2 and "link code 1002 module 103 at 1631..4928" in before["stdout"]
    evidence = dict(schema="fairyc.candidate-project-work-scope/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in names],
        tests=[identity(ROOT / "examples/lexer_candidate/tests" / name) for name in ("module_unit_cleanup_mir.fy", "module_unit_body_ir.fy", "module_project_link.fy")],
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        policy=dict(unitProducer=1048576, projectProducerAndRebinding=4194304, projectScopeExplicit=True,
            cleanupVerificationIndependent=1048576, bodyAndMetadataPhase=1048576, projectIRVerificationAggregate=4194304,
            typeFrameAndMetadataPlans=1048576, capacitiesAndAuthorityChecksUnchanged=True),
        native=dict(report=identity(report_path), cases=3,
            unitWork1048577Rejected=True, projectWork1048577And4194304Accepted=True,
            projectWork4194305Rejected=True, projectCanonicalTypeMutationRejected=True, malformedBodyAndExhaustedBodyWorkRejected=True),
        reusedCExecution=dict(report=identity(execution_path), runs=2, sourceSHA256=generated_hash),
        sourceCheck=identity(check_path),
        originalProject=dict(priorFailure=identity(before_path),
            sourceCount=len(inputs), rawBytes=sum(row["bytes"] for row in inputs),
            inputs={row["path"]: row["sha256"] for row in inputs},
            copiedSourcesDigest=hashlib.sha256(json.dumps(copied, sort_keys=True).encode()).hexdigest(),
            wrapper=identity(PROBE / "src/main.fy"), image=identity(CACHE / "main-link-work-scope-original.exe"),
            imageNote="Exact original image retained before the cache-only diagnostic rebuild",
            sourceCheck=identity(probe_check_path), build=identity(build_path), execution=identity(run_path),
            units=int(compiled[1]), retries=int(compiled[2]), exit=run["exit"], seconds=run["seconds"], stdout=run["stdout"],
            linked=run["exit"] == 0, frames=int(linked[1]) if linked else None, producerWork=int(linked[2]) if linked else None),
        nextFailure=dict(report=identity(diagnostic_path), temporaryCode=510524,
            stage="Owned IR revalidation, aggregate work checkpoint after a body",
            bodyCopyAndCleanupAssemblyPassed=True, projectVerificationAggregateExceeded=True,
            nextTask="Reduce repeated authority lookups during whole-project revalidation"),
        limits=["Validation-only host compilation deadline 120 seconds", "Standalone observer process limit 180 seconds", "Link observation does not establish C emission or original dispatcher execution"],
        deferred=["Original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-work-scope-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, originalLinked=run["exit"] == 0, originalStdout=run["stdout"])))


if __name__ == "__main__":
    main()
