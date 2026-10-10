"""Verify direct merged-layout indexing against repeated units and original inputs."""
import base64
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-link-layout-index-probe"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    report_path = CACHE / "native-project-layout-index.json"
    report = json.loads(report_path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert (report["count"], report["passed"], report["failed"]) == (2, 2, 0)
    cases = {case["identity"]: case for case in report["tests"]}
    stress = "tests::module_project_layout_index::repeated_provider_layouts_keep_indexed_choices_and_field_maps"
    cleanup = "tests::module_project_link::project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    assert set(cases) == {stress, cleanup}
    for case in cases.values():
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    out = base64.b64decode(cases[stress]["stdoutBase64"], validate=True).decode()
    match = re.fullmatch(r"units 128 layouts 512 fields 512 work (\d+)\n", out)
    assert match and int(match[1]) < 65536
    execution_path = CACHE / "project-link-c/execution.json"
    executions = json.loads(execution_path.read_bytes())
    generated_hash = hashlib.sha256(base64.b64decode(cases[cleanup]["stdoutBase64"], validate=True)).hexdigest()
    runs = [row for row in executions["executions"] if row["name"] == cleanup.split("::")[-1]]
    assert executions["passed"] and len(runs) == 2 and {row["profile"] for row in runs} == {"low", "optimized"}
    for row in runs:
        assert row["sourceSHA256"] == generated_hash and row["build"]["exit"] == row["run"]["exit"] == 0
    check_path = CACHE / "check-project-layout-index.json"
    check = json.loads(check_path.read_bytes())
    assert check["exit"] == 0 and "check ok:" in check["stdout"] and not check["stderr"]
    build_path = CACHE / "build-main-link-layout-index-probe.json"
    built = json.loads(build_path.read_bytes())
    probe_check = CACHE / "check-main-link-layout-index-probe.json"
    assert json.loads(probe_check.read_bytes())["exit"] == built["exit"] == 0 and not built["stderr"]
    run_path = CACHE / "run-main-link-layout-index-probe.json"
    run = json.loads(run_path.read_bytes())
    assert run["exit"] in (0, 2) and not run["stderr"]
    counters = re.search(r"preflight type-work (\d+) frame-work (\d+) metadata-work (\d+) linear-work (\d+) layouts (\d+) fields (\d+)\n", run["stdout"])
    assert counters and int(counters[3]) < int(counters[4])
    linked = re.search(r"linked frames (\d+) work (\d+)\n", run["stdout"])
    assert (linked is not None) == (run["exit"] == 0)
    baseline = subprocess.check_output(["git", "show", "0f8a7d75:examples/lexer_candidate/src/module_project_metadata.fy"], cwd=ROOT).decode()
    assert (PROBE / "src/diagnostic_linear_metadata.fy").read_text() == baseline.replace("module module_project_metadata;", "module diagnostic_linear_metadata;")
    copied = []
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert (PROBE / "src" / source.name).read_bytes() == source.read_bytes(), source
            copied.append(identity(source))
    paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    evidence = dict(schema="fairyc.candidate-project-layout-index/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_project_metadata.fy"),
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_layout_index.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"), receiptTool=identity(Path(__file__)),
        policy=dict(canonicalTypeChoiceStoresMergedLayout=True, exactDuplicateComparison=True, extraIndexAllocation=False,
            projectFields=4096, projectLayouts=1024, producerWork=1048576),
        native=dict(report=identity(report_path), cases=2, units=128, providers=16, layouts=512, fields=512, work=int(match[1]),
            checkedMappings=4096, sourceRecipeCoordinatesPreserved=True),
        reusedCExecution=dict(report=identity(execution_path), runs=2, sourceSHA256=generated_hash),
        sourceCheck=identity(check_path),
        originalProject=dict(sourceInputs=[identity(ROOT / path) for path in paths], copiedSources=copied,
            wrapper=identity(PROBE / "src/main.fy"), image=identity(PROBE / "build/main-link-layout-index-probe.exe"),
            baselineCommit="0f8a7d75", control=identity(PROBE / "src/diagnostic_linear_metadata.fy"),
            sourceCheck=identity(probe_check), build=identity(build_path), execution=identity(run_path),
            exit=run["exit"], seconds=run["seconds"], stdout=run["stdout"],
            typeWork=int(counters[1]), frameWork=int(counters[2]), indexedMetadataWork=int(counters[3]),
            linearMetadataWork=int(counters[4]), layouts=int(counters[5]), fields=int(counters[6]),
            linked=run["exit"] == 0, frames=int(linked[1]) if linked else None, producerWork=int(linked[2]) if linked else None),
        limits=["Validation-only host compilation timeout is 120 seconds", "Standalone observer process limit is 180 seconds", "Full link observation does not establish C emission or original dispatcher execution"],
        deferred=["Original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-layout-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=2, stressWork=int(match[1]), indexedMetadataWork=int(counters[3]), linearMetadataWork=int(counters[4]), originalLinked=run["exit"] == 0)))


if __name__ == "__main__":
    main()
