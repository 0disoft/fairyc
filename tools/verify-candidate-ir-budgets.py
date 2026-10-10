"""Bind owned-IR verifier budget regression to original-source observations."""
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


def report(name, profile, passed, failed):
    path = CACHE / name
    data = json.loads(path.read_bytes())
    assert data["engine"] == "native" and data["target"] == "x86_64-windows-msvc" and data["profile"] == profile
    assert (data["count"], data["passed"], data["failed"]) == (passed + failed, passed, failed)
    return data, identity(path)


def main():
    focused, focused_id = report("native-ir-budgets.json", "development", 2, 0)
    body, body_id = report("native-original-bodies-ir-budget.json", "development", 1, 0)
    for case in [*focused["tests"], *body["tests"]]:
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
    stdout = base64.b64decode(body["tests"][0]["stdoutBase64"], validate=True).decode()
    observation = re.fullmatch(r"bodies frames (\d+) producer work (\d+)\n", stdout)
    assert observation and int(observation[1]) > 0 and int(observation[2]) <= 1048576
    after, after_id = report("native-main-project-compile-budget-release.json", "release", 0, 1)
    next_failure = base64.b64decode(after["tests"][0]["stdoutBase64"], validate=True).decode()
    assert next_failure == "stage 6 code 1002 module 213 at 0..0 range 35..36\n"
    assert not base64.b64decode(after["tests"][0]["stderrBase64"], validate=True)
    check = CACHE / "check-ir-budgets.txt"
    assert re.fullmatch(r"check ok: 2 instance\(s\), \d+ source file\(s\), \d+ MIR function\(s\), lock unchanged\n", check.read_text(encoding="utf-8-sig"))
    development_timeout = CACHE / "native-main-project-compile-budget.stderr"
    assert "exceeded 1m0s" in development_timeout.read_text()
    evidence = dict(schema="fairyc.candidate-owned-ir-verifier-budget/v1", passed=True,
        parentCommit=subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_body_ir_verify.fy"),
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_ir_budgets.fy"), receiptTool=identity(Path(__file__)),
        policy=dict(producerWorkCap=1048576, verifierWorkCap=1048576, sdkAuthenticationRetained=True,
                    cleanupVerifierUnchanged=True, traversalAndTableCapsUnchanged=True),
        native=dict(reports=[focused_id, body_id], passedCases=3, originalBodiesFrames=int(observation[1]),
                    originalBodiesProducerWork=int(observation[2]),
                    coverage=["Producer work at the cap can be independently revalidated", "Producer work one over the cap rejects",
                              "Malformed body tables still reject", "Original bodies.fy unit compiles and revalidates with all 235 original main sources"],
                    reusedCases="Boundary/mutation report predates only the added original-file regression; production verifier is identical"),
        check=identity(check), originalCompilationPassed=False,
        original=dict(previousReceipt="docs/ops/evidence/candidate-project-unit-coordination-20261011.json",
                      releaseReport=after_id, nextFailure=next_failure,
                      nextSource="examples/lexer_candidate/src/generic_expand.fy", developmentTimeout=identity(development_timeout)),
        deferred=["Next measured cleanup-lowering budget failure", "Remaining original units", "Original dispatcher linking and C execution"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-owned-ir-verifier-budget-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, originalBodiesFrames=int(observation[1]), producerWork=int(observation[2]), nextSource=evidence["original"]["nextSource"])))


if __name__ == "__main__":
    main()
