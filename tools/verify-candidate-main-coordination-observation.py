"""Bind a standalone Fairy coordination probe to its actual build/run outcomes."""
import hashlib
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROJECT = CACHE / "main-coordination-full-probe"


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def main():
    checked = json.loads((CACHE / "check-main-coordination-probe.json").read_bytes())
    built = json.loads((CACHE / "build-main-coordination-probe.json").read_bytes())
    run = json.loads((CACHE / "run-main-coordination-probe.json").read_bytes())
    assert checked["exit"] == built["exit"] == 0 and not checked["stderr"] and not built["stderr"]
    assert "lock verified" in built["stdout"] and "release" in built["stdout"]
    sources = ROOT / "examples/lexer_candidate/src"
    copied = []
    for path in sources.glob("*.fy"):
        if path.name == "main.fy":
            continue
        assert (PROJECT / "src" / path.name).read_bytes() == path.read_bytes(), path
        copied.append(identity(path))
    wrapper = (PROJECT / "src/main.fy").read_text(encoding="utf-8")
    assert "compile::build(ref sources,ref metadata)" in wrapper
    assert "compile::range(" not in wrapper and "@test" not in wrapper
    paths = json.loads((CACHE / "main-project-liveness-inputs.json").read_bytes())
    assert run["argv"][1:] == [str((ROOT / path).resolve()) for path in paths]
    assert run["timeoutSeconds"] == 180
    timed_report = CACHE / "native-main-owned-body-expression-cap.json"
    timed_stderr = CACHE / "native-main-owned-body-expression-cap.stderr"
    assert timed_report.read_bytes() == b""
    assert "MSVC target executable exceeded 1m0s" in timed_stderr.read_text()
    if run["exit"] == 0:
        assert re.fullmatch(r"compiled units \d+ retries \d+\n", run["stdout"])
        assert not run["timedOut"] and not run["stderr"]
    evidence = dict(schema="fairyc.candidate-main-coordination-observation/v1",
        originalCompilationPassed=run["exit"] == 0,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        receiptTool=identity(Path(__file__)),
        wrapper=identity(PROJECT / "src/main.fy"), image=identity(PROJECT / "build/main-coordination-probe.exe"),
        copiedSources=copied, sourceInputs=[identity(ROOT / path) for path in paths],
        sourceCheck=identity(CACHE / "check-main-coordination-probe.json"),
        build=identity(CACHE / "build-main-coordination-probe.json"),
        execution=dict(report=identity(CACHE / "run-main-coordination-probe.json"), exit=run["exit"],
            seconds=run["seconds"], timeoutSeconds=180, timedOut=run["timedOut"], stdout=run["stdout"], stderr=run["stderr"]),
        standardNativeTest=dict(passed=False, executionLimitSeconds=60,
            stdout=identity(timed_report), stderr=identity(timed_stderr)),
        limits=["Standalone validation probe uses the existing Fairy coordinator implementation",
            "180-second observer deadline does not change the standard native test's 60-second deadline",
            "Build uses the existing validation-only 120-second host compilation overlay",
            "A successful observation does not establish default-deadline performance or dispatcher execution"],
        deferred=["Default compilation timeout/performance", "Original dispatcher linking and C execution", "Native driver", "Linux", "Standalone compiler profiles", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-main-coordination-observation-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(originalCompilationPassed=evidence["originalCompilationPassed"], seconds=run["seconds"], stdout=run["stdout"], standardNativeTestPassed=False)))


if __name__ == "__main__":
    main()
