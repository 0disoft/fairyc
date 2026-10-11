"""Bind bounded preparation histories to the complete original C bundle."""
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
CACHE = ROOT / ".cache/candidate-tagged/sdk-c-execution"
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("parts", ROOT / "tools/verify-candidate-project-c-parts.py")
parts = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parts)


def identity(path):
    data = path.read_bytes()
    return dict(path=path.relative_to(ROOT).as_posix(), bytes=len(data), sha256=hashlib.sha256(data).hexdigest())


def read(name):
    return json.loads((CACHE / name).read_bytes())


def main():
    native = read("native-c-entry-scopes.json")
    assert native["engine"] == "native" and native["profile"] == "release"
    assert native["count"] == native["passed"] == 2 and native["failed"] == 0
    before = read("native-c-entry-parts.json")
    def outputs(report):
        assert all(test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True) for test in report["tests"])
        return {test["identity"]: base64.b64decode(test["stdoutBase64"], validate=True) for test in report["tests"]}
    assert outputs(native) == outputs(before)
    prior_execution = read("c-entry-parts-native-c/execution.json")
    assert prior_execution["passed"] and sum(len(case["executions"]) for case in prior_execution["cases"].values()) == 4
    check = read("check-c-scopes-final.json")
    assert check["exit"] == 0 and "291 source file(s), 3653 MIR function(s), lock verified" in check["stdout"] and not check["stderr"]
    meter = read("run-main-c-entry-meter.json")
    assert meter["exit"] == 2 and not meter["timedOut"] and not meter["stderr"]
    assert "reachability work 787768 bytes 0 extra 3222\n" in meter["stdout"]
    assert "aggregate types work 1016330 bytes 107055 extra 0\nwriter failure work 1048576 bytes 107055 extra 1002\n" in meter["stdout"]
    original = read("run-main-c-entry-scopes.json")
    assert original["exit"] == 0 and not original["timedOut"] and not original["stderr"]
    assert original["stdout"] == "compiled units 48 retries 25\nlinked frames 3558 work 3321980\nproject drops 6392 root drops 4\nroot CFG blocks 247 work 7650\npartition frames 3558 units 28 retries 0 work 3233248\npacket header 655466 literals 149489\n"
    prior_path = ROOT / "docs/ops/evidence/candidate-project-c-parts-20261011.json"
    inputs = json.loads(prior_path.read_bytes())["originalProject"]["sourceInputs"]
    assert len(inputs) == 236 and original["argv"][1:] == meter["argv"][1:] == [source["path"] for source in inputs]
    assert all(identity(ROOT / source["path"]) == source for source in inputs)
    probe = CACHE / "main-c-entry-scopes"
    for source in (ROOT / "examples/lexer_candidate/src").glob("*.fy"):
        if source.name != "main.fy":
            assert source.read_bytes() == (probe / "src" / source.name).read_bytes(), source
    packet_path = CACHE / "project-entry-scopes-packet.bin"
    header, literals, units, frames, pool = parts.packet(packet_path.read_bytes(), False)
    assert (len(header), len(literals), len(units), frames, pool) == (655466, 149489, 28, 3222, 775)
    assert sum(map(len, units)) == 19288554 and max(map(len, units)) == 1432228
    evidence = dict(schema="fairyc.candidate-c-preparation-scopes/v1", passed=True,
        implementation=identity(ROOT / "examples/lexer_candidate/src/module_unit_c_emit.fy"),
        contract=identity(ROOT / "docs/architecture/candidate-project-c-parts-v0.md"), receiptTool=identity(Path(__file__)),
        sourceCheck=check, native=dict(report=identity(CACHE / "native-c-entry-scopes.json"), passed=2),
        unchangedPackets=dict(exact=True, previousNative=identity(CACHE / "native-c-entry-parts.json"), execution=identity(CACHE / "c-entry-parts-native-c/execution.json"), reusedExecutions=4),
        failureMeter=dict(report=identity(CACHE / "run-main-c-entry-meter.json"), wrapper=identity(CACHE / "main-c-entry-meter/src/module_unit_c_emit.fy"), exit=2, stdout=meter["stdout"]),
        originalProject=dict(emissionPassed=True, sourceCount=236, sourceIdentityReceipt=identity(prior_path), inputsMatch=True, report=identity(CACHE / "run-main-c-entry-scopes.json"), exit=0, seconds=original["seconds"], stdout=original["stdout"], wrapper=identity(probe / "src/main.fy"), image=identity(probe / "build/main-c-entry-scopes.exe"), packet=identity(packet_path), frames=3558, selectedFrames=3222, units=28, retries=0, work=3233248, headerBytes=len(header), literalBytes=len(literals), literalDefinitions=775, bodyBytes=sum(map(len, units)), maxBodyBytes=max(map(len, units))),
        limits=dict(analysisWork=1048576, commonDeclarationWork=1048576, unitWork=1048576, outputBytesPerBuffer=2097152, projectWork=4194304),
        deferred=["Original executable linkage and execution", "Native driver/publication", "Default timeout/performance", "Standalone compiler profiles", "Linux", "Full suite", "Stage1/Stage2"], pushed=False)
    (ROOT / "docs/ops/evidence/candidate-c-preparation-scopes-20261011.json").write_text(json.dumps(evidence, indent=2) + "\n")
    print("PASS: complete original C emission, 2 native cases, 4 reused C executions")


if __name__ == "__main__":
    main()
