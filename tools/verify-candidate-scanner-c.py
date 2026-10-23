"""Execute an original-scanner artifact with the actual Windows runtime."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("candidate_c", ROOT / "tools/verify-candidate-module-c.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--report", type=Path, required=True)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--reference", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--vcvars", type=Path, required=True)
    args = parser.parse_args()
    report = json.loads(args.report.read_bytes())
    assert report["engine"] == "native" and report["target"] == "x86_64-windows-msvc"
    assert report["count"] == report["passed"] == 1 and report["failed"] == 0
    test = report["tests"][0]
    assert test["identity"] == "tests::module_unit_c_scanner::sdk_c_executes_original_scanner_report"
    stdout = base64.b64decode(test["stdoutBase64"], validate=True).decode()
    assert not base64.b64decode(test["stderrBase64"], validate=True)
    proof = re.fullmatch(r"retained instances (\d+)\nC bytes (\d+) SHA256 words((?: \d+){8})\n", stdout)
    assert proof, stdout
    source = args.source.read_bytes()
    assert source.startswith(b"/* fairyc.candidate-module-owned-c/v1 */\n")
    digest = b"".join(int(word).to_bytes(4, "big") for word in proof[3].split()).hex()
    assert len(source) == int(proof[2]) and hashlib.sha256(source).hexdigest() == digest
    assert 64 < int(proof[1]) <= 256
    reference = json.loads(args.reference.read_bytes())
    assert reference["passed"] and reference["runs"][-1]["exit"] == 0
    expected = reference["runs"][-1]["stdout"]
    assert reference["runs"][-1]["stderr"] == ""
    vcvars = str(args.vcvars.resolve())
    assert args.vcvars.is_file() and not any(c in vcvars for c in '\r\n"&|<>^%!')
    setup = driver.run(f'cmd.exe /d /s /c ""{vcvars}" x64 >nul && set"', ROOT)
    assert setup["exit"] == 0, setup
    environment = {key.upper(): value for key, value in os.environ.items()}
    for line in setup["stdout"].splitlines():
        if "=" in line and not line.startswith("="):
            key, value = line.split("=", 1)
            environment[key.upper()] = value
    compiler = next(Path(folder) / "cl.exe" for folder in environment["PATH"].split(";")
                    if (Path(folder) / "cl.exe").is_file())
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    generated = output / "scanner.c"
    generated.write_bytes(source)
    components = [ROOT / "internal/cstdlib" / name for name in
                  ("stage0_system_heap.c", "stage0_buffer.c", "stage0_vector_storage.c", "stage0_windows_console.c")]
    runtime = output / "scanner-runtime.c"
    prefix = b"""#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <stdio.h>
static size_t attempts,live;
static unsigned buffer_drops,vector_releases,errors;
static void *test_malloc(size_t size){++attempts;void *p=malloc(size);if(p)++live;return p;}
static void test_free(void *p){if(p){if(!live)abort();--live;}free(p);}
#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) test_malloc(size)
#define FY_STD_SYSTEM_HEAP_FREE_V0(raw) test_free(raw)
#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)
#define fy_std_buffer_drop_v0 test_actual_buffer_drop
#define fy_private_vector_release_empty_v0 test_actual_vector_release
"""
    suffix = rb"""
#undef fy_std_buffer_drop_v0
#undef fy_private_vector_release_empty_v0
uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *b){uint32_t s=test_actual_buffer_drop(b);++buffer_drops;if(s)++errors;return s;}
uint32_t fy_private_vector_release_empty_v0(void *p){uint32_t s=test_actual_vector_release(p);++vector_releases;if(s)++errors;return s;}
extern int32_t fy_candidate_entry(void);
int main(void){int32_t s=fy_candidate_entry();fprintf(stderr,"{\"heapAttempts\":%zu,\"heapLive\":%zu,\"bufferDrops\":%u,\"vectorReleases\":%u,\"cleanupErrors\":%u}\n",attempts,live,buffer_drops,vector_releases,errors);return s?s:(live||errors?90:0);}
"""
    runtime.write_bytes(prefix + b"\n".join(path.read_bytes() for path in components) + suffix)
    trap = output / "trap.c"
    trap.write_text("#include <stdint.h>\n#include <stdlib.h>\n_Noreturn void fy_runtime_trap(uint32_t category){exit(100+(int)category); }\n" + driver.LOCATION_TRAP_SHIM, encoding="utf-8")
    executions = []
    for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
        binary = output / f"scanner-{profile}.exe"
        build = driver.run([str(compiler), "/nologo", "/std:c11", "/TC", flag,
                            "/DFY_CANDIDATE_NO_MAIN", str(generated), str(runtime), str(trap),
                            f"/Fe:{binary}", "/link", "/INCREMENTAL:NO"], output, environment)
        assert build["exit"] == 0, build
        run = driver.run([str(binary)], output, environment)
        assert run["exit"] == 0 and run["stdout"] == expected, run
        counts = json.loads(run["stderr"])
        assert counts["heapLive"] == counts["cleanupErrors"] == 0
        assert counts["vectorReleases"] > 0 and counts["heapAttempts"] > 0
        executions.append(dict(profile=profile, sourceSHA256=digest, counts=counts, build=build, run=run))
    assert executions[0]["counts"] == executions[1]["counts"]
    receipt = dict(schema="fairyc.candidate-original-scanner-c-execution/v1", passed=True,
                   retainedInstances=int(proof[1]), sourceBytes=len(source), sourceSHA256=digest,
                   executions=executions, runtimeInputs=[dict(path=str(p.relative_to(ROOT)),
                   sha256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in components])
    (output / "receipt.json").write_text(json.dumps(receipt, indent=2) + "\n", encoding="utf-8")
    print(f"Original scanner C execution passed: {len(executions)} runs")


if __name__ == "__main__":
    main()
