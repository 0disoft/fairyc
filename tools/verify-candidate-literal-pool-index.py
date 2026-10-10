"""Bind indexed literal collection to native, generated-C and original-input evidence."""
import base64
import hashlib
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
PROBE = CACHE / "main-c-phase-literal-index-probe"


def identity(path):
    raw = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())


def read(name):
    return json.loads((CACHE / name).read_bytes())


def main():
    native = read("native-literal-pool-index.json")
    assert (native["count"], native["passed"], native["failed"]) == (3, 3, 0)
    assert native["engine"] == "native" and native["profile"] == "release"
    cases = {}
    for case in native["tests"]:
        assert case["outcome"] == "passed" and not base64.b64decode(case["stderrBase64"], validate=True)
        cases[case["identity"].split("::")[-1]] = base64.b64decode(case["stdoutBase64"], validate=True)
    # Compute the expected full storage blob independently of Fairy's SHA and sorting.
    values = [f"literal_{i}".encode() for i in range(800)]
    expected = b"".join(b"static const uint8_t fy_unit_literal_" + digest.encode() + b"[]={" +
        b",".join(str(byte).encode() for byte in data) + b"};\n"
        for digest, data in sorted((hashlib.sha256(data).hexdigest(), data) for data in values))
    digest = hashlib.sha256(expected).hexdigest()
    stress = cases["literal_pool_indexes_duplicates_and_sorts_large_unique_sets"]
    assert stress == f"work 75396 bytes {len(expected)}\nsha256 {digest}\n".encode()
    for name, data in cases.items():
        if name.startswith("literal_c_"):
            storage = re.findall(rb"static const uint8_t fy_unit_literal_([0-9a-f]{64})\[\]=\{([0-9,]+)\};", data)
            hashes = [hash_ for hash_, _ in storage]
            assert hashes == sorted(set(hashes))
            for hash_, payload in storage:
                raw = bytes(int(byte) for byte in payload.split(b","))
                assert hashlib.sha256(raw).hexdigest().encode() == hash_ or (
                    raw == b"\0" and hashlib.sha256(b"").hexdigest().encode() == hash_)
    executed = read("literal-pool-index-c/execution.json")
    assert executed["passed"] and len(executed["executions"]) == 4
    assert executed["nativeReportSHA256"] == identity(CACHE / "native-literal-pool-index.json")["sha256"]
    for run in executed["executions"]:
        assert run["sourceSHA256"] == hashlib.sha256(cases[run["name"]]).hexdigest()
        assert run["build"]["exit"] == run["run"]["exit"] == 0
        assert not run["run"]["stdout"] and not run["run"]["stderr"]
    check = read("check-literal-pool-index-final.json")
    assert check["exit"] == 0 and "lock verified" in check["stdout"] and not check["stderr"]
    before = read("run-main-literal-budget-diagnostic-probe.json")
    assert before["exit"] == 2 and "emit code 731003" in before["stdout"]
    assert "while(position<v::len<Literal>" in read("literal-budget-diagnostic-codes.json")["731003"]
    after = read("run-main-c-phase-literal-index-probe.json")
    assert after["exit"] == 2 and not after["timedOut"] and not after["stderr"]
    assert "compiled units 48 retries 25\nlinked frames 3558 work 3321980\n" in after["stdout"]
    assert "literal definitions code 0 work 386674 bytes 154845\n" in after["stdout"]
    assert after["stdout"].endswith("emit code 721007 module 0 at 0..0 C bytes 0\n")
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name not in ("main.fy", "module_unit_c_emit.fy"):
            assert source.read_bytes() == (PROBE / "src" / source.name).read_bytes(), source
    paths = read("main-cfg-drop-index-closed-inputs.json")
    assert len(paths) == len(set(paths)) == 236
    assert after["argv"][1:] == paths
    modules = {re.search(r"^module\s+([^;]+);", (ROOT / path).read_text(), re.M).group(1): path for path in paths}
    seen, pending = set(), ["main"]
    while pending:
        module = pending.pop()
        if module in seen:
            continue
        seen.add(module)
        pending.extend(re.findall(r"^import\s+([^\s]+)\s+as", (ROOT / modules[module]).read_text(), re.M))
    assert len(seen) == len(paths)
    invalid = read("native-literal-pool-index-output-cap-attempt.json")
    assert invalid["exit"] == 6 and "exceeded 65536 output bytes" in invalid["stderr"]
    evidence = dict(schema="fairyc.candidate-literal-pool-index/v1", passed=True,
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_c_literals.fy"),
        test=identity(ROOT / "examples/lexer_candidate/tests/module_unit_c_literals.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-compilation-units-v0.md"),
        receiptTool=identity(Path(__file__)), sourceCheck=check,
        native=dict(report=identity(CACHE / "native-literal-pool-index.json"), passed=3,
            stress=dict(unique=800, occurrences=1600, work=75396, bytes=len(expected), sha256=digest)),
        generatedC=dict(report=identity(CACHE / "literal-pool-index-c/execution.json"), executions=4, profiles=["/Od", "/O2"]),
        originalProject=dict(inputs=[identity(ROOT / path) for path in paths], exactRootImportClosure=True,
            count=236, units=48, retries=25, frames=3558, fullLinkPassed=True,
            standaloneLiteralDefinitions=dict(passed=True, work=386674, bytes=154845), cGenerationPassed=False,
            before=identity(CACHE / "run-main-literal-budget-diagnostic-probe.json"),
            after=identity(CACHE / "run-main-c-phase-literal-index-probe.json"),
            nextPhase="Literal definitions with work shared by preceding C stages", transactionalOutputBytes=0),
        invalidHarnessAttempt=dict(observation=invalid, reason="Full C stdout exceeded native capture cap; corrected test asserts independent full-blob SHA internally"),
        limits=dict(work=1048576, outputBytes=2097152, raised=False),
        deferred=["Full original C generation/execution", "Native driver", "Default timeout/performance", "Standalone candidate profiles", "Linux", "Full suite", "Stage1/Stage2"],
        stableCLIChanged=False, sdkABIChanged=False, GoChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    shared_path = CACHE / "run-main-c-literal-shared-budget-probe.json"
    if shared_path.exists():
        shared = json.loads(shared_path.read_bytes())
        assert shared["exit"] == 2 and not shared["timedOut"] and not shared["stderr"]
        assert shared["stdout"].endswith("emit code 721007 module 729542 at 1048576..106977 C bytes 0\n")
        evidence["originalProject"]["sharedBudgetDiagnostic"] = dict(
            report=identity(shared_path), initialWork=729542, exhaustedWork=1048576,
            temporaryOutputBytes=106977, transactionalOutputBytes=0,
            reason="Preceding stages leave 319034 work; standalone literal definitions require 386674")
    (ROOT / "docs/ops/evidence/candidate-literal-pool-index-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(dict(passed=True, nativePassed=3, generatedCExecutions=4, originalC=False)))


if __name__ == "__main__":
    main()
