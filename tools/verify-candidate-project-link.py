"""Record independent-unit linking and actual generated-C executions."""
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


def native(path, count):
    report = json.loads(path.read_bytes())
    assert report["engine"] == "native" and report["profile"] == "release"
    assert report["target"] == "x86_64-windows-msvc"
    assert report["count"] == report["passed"] == count and report["failed"] == 0
    for test in report["tests"]:
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
    return report


def main():
    report_path = CACHE / "native-project-link.json"
    retry_path = CACHE / "native-project-shared-cleanup.json"
    report = native(report_path, 4)
    retry = native(retry_path, 1)
    prefix = "tests::module_project_link::"
    cases = {test["identity"].removeprefix(prefix): test for test in report["tests"]}
    retry_name = "project_c_coalesces_shared_cleanup_bodies_and_preserves_drops"
    assert retry["tests"][0]["identity"] == prefix + retry_name
    cases[retry_name] = retry["tests"][0]
    expected = {"project_c_links_three_ordinary_units_with_different_type_graphs": "",
                "project_c_namespaces_generic_instances_from_distinct_units": "", retry_name: "7,"}
    assert set(cases) == set(expected) | {"project_link_rejects_missing_and_duplicate_ordinary_bodies"}
    execution_path = CACHE / "project-link-c/execution.json"
    execution = json.loads(execution_path.read_bytes())
    assert execution["passed"] and execution["suite"] == "project" and len(execution["executions"]) == 6
    assert execution["nativeReport"]["sha256"] == identity(report_path)["sha256"]
    assert execution["nativeReruns"][0]["sha256"] == identity(retry_path)["sha256"]
    observed = set()
    binaries = []
    for item in execution["executions"]:
        name, profile = item["name"], item["profile"]
        assert name in expected and profile in {"low", "optimized"}
        assert (name, profile) not in observed
        observed.add((name, profile))
        source = base64.b64decode(cases[name]["stdoutBase64"], validate=True)
        assert source.startswith(b"/* fairyc.candidate-module-owned-c/v1 */\n")
        assert item["sourceSHA256"] == hashlib.sha256(source).hexdigest()
        assert item["build"]["exit"] == item["run"]["exit"] == 0
        assert item["run"]["stdout"] == expected[name] and not item["run"]["stderr"]
        binaries.append(identity(Path(item["run"]["argv"][0])))
    assert len(observed) == 6
    check_path = CACHE / "check-project-link.txt"
    locked_path = CACHE / "check-project-link-locked.txt"
    checked = re.fullmatch(r"check ok: 2 instance\(s\), (\d+) source file\(s\), (\d+) MIR function\(s\), lock verified\n", locked_path.read_text(encoding="utf-8-sig"))
    assert checked and "check ok:" in check_path.read_text(encoding="utf-8-sig")
    names = ("module_project_link.fy", "module_project_frames.fy", "module_project_metadata.fy", "module_project_body.fy",
             "module_project_body_equal.fy", "module_project_cleanup.fy", "module_project_calls.fy", "module_project_types.fy")
    evidence = dict(schema="fairyc.candidate-project-body-linking/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        sources=[identity(ROOT / "examples/lexer_candidate/src" / name) for name in names],
        test=identity(ROOT / "examples/lexer_candidate/tests/module_project_link.fy"),
        receiptTool=identity(Path(__file__)), executionTool=identity(ROOT / "tools/verify-candidate-module-c.py"),
        native=dict(report=identity(report_path), uniquePassed=4, sharedCleanupFieldRebinding=identity(retry_path),
                    coverage=["Three separately bound/typed/moved/lowered units execute ordinary cross-unit calls",
                              "Generic instances with unit-local ID 1 receive separate global frame identities",
                              "Shared resource cleanup bodies coalesce only after exact normalized body equality",
                              "Different field-table positions rebind to the same owner field; cleanup runs exactly once",
                              "Missing peer bodies and duplicate ordinary bodies reject before C output"]),
        generatedC=dict(receipt=identity(execution_path), executions=6, profiles=["/Od", "/O2"], binaries=binaries,
                        expectedCleanupStdout="7,"),
        sourceCheck=dict(check=identity(check_path), locked=identity(locked_path), sourceFiles=int(checked[1]), mirFunctions=int(checked[2])),
        limits=dict(sourceUnits=256, canonicalTypes=4096, frames=1024, layouts=1024, fields=1024, callables=1024, work=1048576,
                    note="Existing owned-IR budgets remain; complete original dispatcher admission is not proven here"),
        bootstrap=dict(image=identity(ROOT / ".cache/candidate-tagged/compiler-file-tokens-120s.exe"),
                       hostCompileTimeoutSeconds=120, nativeExecutionTimeoutSeconds=60),
        deferred=["Compile and link all original dispatcher units; adjust project budgets from exact failures",
                  "Whole-dispatcher C generation/execution", "Native driver integration", "Default compilation timeout/performance",
                  "Standalone candidate development/Release", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-project-body-linking-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=4, generatedCRuns=6, cleanupStdout="7,")))


if __name__ == "__main__":
    main()
