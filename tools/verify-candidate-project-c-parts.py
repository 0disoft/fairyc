"""Compile and execute length-delimited, private Fairy project C bundles."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("candidate_c", ROOT / "tools/verify-candidate-module-c.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)

CASES = {
    "project_c_partition_links_shared_nominal_cleanup_across_units": (3, "7,"),
    "project_c_partition_keeps_generic_instance_namespaces": (5, ""),
    "project_c_partition_preserves_literal_pointer_identity": (4, ""),
}
ENTRY_CASE = "project_c_entry_partition_keeps_nested_result_cleanup_and_omits_unused_bodies"


def packet(data, complete=True):
    assert data.startswith(b"FYCPARTS1\n")
    offset = len(b"FYCPARTS1\n")

    def number():
        nonlocal offset
        end = data.index(b"\n", offset)
        value = data[offset:end]
        assert re.fullmatch(rb"0|[1-9][0-9]{0,7}", value), value
        offset = end + 1
        return int(value)

    def text():
        nonlocal offset
        size = number()
        assert 0 < size <= 2097152 and size <= len(data) - offset
        value = data[offset:offset + size]
        offset += size
        return value

    header, literals = text(), text()
    count = number()
    assert 0 < count <= 4096
    units = [text() for _ in range(count)]
    assert offset == len(data), "Trailing or truncated packet bytes"
    assert header.startswith(b"#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n") and header.endswith(b"#endif\n")
    assert literals.startswith(b'/* fairyc.candidate-project-literals-c/v1 */\n#include "fy_project.h"\n')
    assert all(unit.startswith(b'/* fairyc.candidate-project-unit-c/v1 */\n#include "fy_project.h"\n') for unit in units)
    declarations = re.findall(rb"extern const uint8_t fy_unit_literal_([0-9a-f]{64})\[\];", header)
    definitions = re.findall(rb"^const uint8_t fy_unit_literal_([0-9a-f]{64})\[\]=\{([0-9,]+)\};$", literals, re.M)
    assert declarations == [digest for digest, _ in definitions] == sorted(set(declarations))
    assert b"static const uint8_t fy_unit_literal_" not in header
    assert all(b"const uint8_t fy_unit_literal_" not in unit for unit in units)
    for digest, payload in definitions:
        raw = bytes(int(value) for value in payload.split(b","))
        assert hashlib.sha256(raw).hexdigest().encode() == digest or (raw == b"\0" and hashlib.sha256(b"").hexdigest().encode() == digest)
    frames = [int(frame) for unit in units for frame in re.findall(rb"^extern [^\n]+ fy_fn_([0-9]+)\([^\n]*\)\{", unit, re.M)]
    assert frames == sorted(set(frames)) and frames and frames[-1] <= 8192
    if complete:
        assert frames == list(range(1, len(frames) + 1)), "Every frame must be defined once in original order"
    assert sum(unit.count(b"int32_t fy_candidate_entry(void){") for unit in units) == 1
    return header, literals, units, len(frames), len(definitions)


def environment(vcvars):
    value = str(vcvars.resolve())
    assert vcvars.is_file() and not any(c in value for c in '\r\n"&|<>^%!')
    setup = driver.run(f'cmd.exe /d /s /c ""{value}" x64 >nul && set"', ROOT)
    assert setup["exit"] == 0, setup
    env = {key.upper(): value for key, value in os.environ.items()}
    for line in setup["stdout"].splitlines():
        if "=" in line and not line.startswith("="):
            key, value = line.split("=", 1)
            env[key.upper()] = value
    compiler = next(Path(folder) / "cl.exe" for folder in env["PATH"].split(";") if (Path(folder) / "cl.exe").is_file())
    return compiler, env


def execute(data, folder, compiler, env, expected_frames, expected_stdout, complete=True):
    header, literals, units, frames, pool_size = packet(data, complete)
    assert frames == expected_frames
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "fy_project.h").write_bytes(header)
    sources = [folder / "fy_literals.c", *[folder / f"fy_unit_{i}.c" for i in range(len(units))]]
    for path, source in zip(sources, [literals, *units]):
        path.write_bytes(source)
    shim = folder / "trap.c"
    shim.write_text('#include <stdint.h>\n#include <stdio.h>\n#include <stdlib.h>\nvoid observe_cleanup(int32_t n){printf("%d,",(int)n);}\n_Noreturn void fy_runtime_trap(uint32_t n){exit(100+(int)n);}\n')
    observations = []
    for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
        binary = folder / f"project-{profile}.exe"
        built = driver.run([str(compiler), "/nologo", "/std:c11", "/TC", flag, *map(str, sources), str(shim), f"/Fe:{binary}", "/link", "/INCREMENTAL:NO"], folder, env)
        assert built["exit"] == 0, built
        result = driver.run([str(binary)], folder)
        assert result["exit"] == 0 and result["stdout"] == expected_stdout and not result["stderr"], result
        observations.append(dict(profile=profile, build=built, run=result))
    return dict(frames=frames, units=len(units), literalDefinitions=pool_size, headerBytes=len(header), sourceBytes=sum(map(len, [literals, *units])), packetSHA256=hashlib.sha256(data).hexdigest(), executions=observations)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", required=True, type=Path)
    parser.add_argument("--vcvars", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--suite", choices=("partition", "entry"), default="partition")
    args = parser.parse_args()
    native = json.loads(args.report.read_bytes())
    assert native["engine"] == "native" and native["profile"] == "release" and native["target"] == "x86_64-windows-msvc"
    assert native["count"] == native["passed"] == (4 if args.suite == "partition" else 2) and native["failed"] == 0
    cases = {}
    for test in native["tests"]:
        assert test["outcome"] == "passed" and not base64.b64decode(test["stderrBase64"], validate=True)
        name = test["identity"].split("::")[-1]
        assert name not in cases
        cases[name] = base64.b64decode(test["stdoutBase64"], validate=True)
    stress = b""
    expected = CASES
    if args.suite == "partition":
        stress = cases.pop("literal_pool_indexes_duplicates_and_sorts_large_unique_sets")
        assert re.fullmatch(rb"work \d+ bytes \d+\nsha256 865a689f953e2a39bad216b2f9863d4f80058be03c5e03994834396ed3f0df48\n", stress)
    else:
        expected = {ENTRY_CASE: (4, "99,7,"), "project_c_partition_preserves_literal_pointer_identity": CASES["project_c_partition_preserves_literal_pointer_identity"]}
    assert cases.keys() == expected.keys()
    compiler, env = environment(args.vcvars)
    output = args.output.resolve()
    results = {}
    for name, (frames, stdout) in expected.items():
        if name == ENTRY_CASE:
            header, _, units, selected, _ = packet(cases[name], False)
            assert len(units) == 6 and selected == 4
            assert not any(b"unused_cleanup(" in unit for unit in units)
            assert len(re.findall(rb"^static void fy_unit_drop_[0-9]+\([^\n]*\)\{", header, re.M)) == 3
        results[name] = execute(cases[name], output / name, compiler, env, frames, stdout, name != ENTRY_CASE)
        if name == "project_c_partition_preserves_literal_pointer_identity":
            assert results[name]["literalDefinitions"] == 3
        print(f"PASS /Od and /O2: {name}", flush=True)
    (output / "execution.json").write_text(json.dumps(dict(schema="fairyc.candidate-project-c-parts-execution/v1", passed=True, compiler=str(compiler), nativeReport=str(args.report), literalLegacyStress=stress.decode(), cases=results), indent=2) + "\n")


if __name__ == "__main__":
    main()
