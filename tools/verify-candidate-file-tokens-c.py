"""Run the unchanged file-token path through owned C and the Go reference image."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("scanner_probe", ROOT / "tools/verify-candidate-scanner-c.py")
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)
driver = probe.driver


def fixtures(folder):
    data = {
        "empty.fy": b"",
        "source-한글-λ.fy": 'module main;\r\n/// doc\r\nfn i32 main(){let ref str s="한\\n";return 7;}\n'.encode(),
        "literals.fy": b'let a=0xFF;let b=1_234;let f=1.25e+2;let t=true;let n=null;let c=c"ok";let b=b"A\\0\\xff";let s="\\u{03bb}";',
        "comments.fy": b'\xef\xbb\xbf// line\r\n/* outer /* inner */ end */\tlet n=3;',
        "invalid-byte.fy": b'let n=1;\xff',
        "bad-utf8-string.fy": b'let s="\xff";',
        "bad-number.fy": b'let n=0xGG;',
        "unclosed-string.fy": b'module main;"unterminated',
        "unclosed-comment.fy": b'let n=1;/*tail',
        "bad-cstring.fy": b'let c=c"a\\0b";',
        "newline-diagnostics.fy": b'let n=1;\r\n@?\r#',
        "exact.fy": b" " * 65536,
        "overflow.fy": b"A" * 65537,
    }
    for name, raw in data.items():
        (folder / name).write_bytes(raw)
    return data


def binary_run(binary, argv, cwd, environment):
    run = subprocess.run([str(binary), *argv], cwd=cwd, env=environment,
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
    return dict(argv=[str(binary), *argv], exit=run.returncode,
                stdout=run.stdout.decode("utf-8"), stderr=run.stderr.decode("utf-8"))


def main():
    parser = argparse.ArgumentParser()
    for name in ("report", "source", "reference", "output", "vcvars"):
        parser.add_argument("--" + name, type=Path, required=True)
    args = parser.parse_args()
    native = json.loads(args.report.read_bytes())
    assert native["count"] == native["passed"] == 1 and native["failed"] == 0
    assert native["engine"] == "native" and native["target"] == "x86_64-windows-msvc"
    test = native["tests"][0]
    assert test["identity"] == "tests::module_unit_c_file_tokens::sdk_c_executes_original_file_tokens"
    assert not base64.b64decode(test["stderrBase64"], validate=True)
    proof = re.fullmatch(r"retained instances (\d+)\nC bytes (\d+) SHA256 words((?: \d+){8})\n",
                         base64.b64decode(test["stdoutBase64"], validate=True).decode())
    assert proof
    source = args.source.read_bytes()
    digest = b"".join(int(word).to_bytes(4, "big") for word in proof[3].split()).hex()
    assert len(source) == int(proof[2]) and hashlib.sha256(source).hexdigest() == digest
    assert source.startswith(b"/* fairyc.candidate-module-owned-c/v1 */\n") and 64 < int(proof[1]) <= 256
    assert args.reference.is_file()
    vcvars = str(args.vcvars.resolve())
    assert args.vcvars.is_file() and not any(c in vcvars for c in '\r\n"&|<>^%!')
    setup = driver.run(f'cmd.exe /d /s /c ""{vcvars}" x64 >nul && set"', ROOT)
    assert setup["exit"] == 0, setup
    environment = {key.upper(): value for key, value in os.environ.items()}
    for line in setup["stdout"].splitlines():
        if "=" in line and not line.startswith("="):
            key, value = line.split("=", 1)
            environment[key.upper()] = value
    environment.pop("FY_TEST_HEAP_FAIL_AT", None)
    environment.pop("FY_TEST_RECEIPT", None)
    compiler = next(Path(folder) / "cl.exe" for folder in environment["PATH"].split(";")
                    if (Path(folder) / "cl.exe").is_file())
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    generated = output / "file-tokens.c"
    generated.write_bytes(source)
    components = [ROOT / "internal/cstdlib" / name for name in
                  ("stage0_system_heap.c", "stage0_buffer.c", "stage0_vector_storage.c",
                   "stage0_windows_console.c", "stage0_text_view.c", "stage0_windows_file.c",
                   "stage0_windows_file_portable.c", "stage0_windows_process_args.c")]
    runtime = output / "runtime.c"
    prefix = b"""#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <stdio.h>
static size_t attempts,live;
static unsigned buffer_drops,vector_releases,errors;
static unsigned file_opened,file_closed,file_dropped,file_auto_closed;
static void *test_malloc(size_t size){++attempts;const char *failure=getenv("FY_TEST_HEAP_FAIL_AT");if(failure&&strtoul(failure,NULL,10)==attempts)return NULL;void *p=malloc(size);if(p)++live;return p;}
static void test_free(void *p){if(p){if(!live)abort();--live;}free(p);}
#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) test_malloc(size)
#define FY_STD_SYSTEM_HEAP_FREE_V0(raw) test_free(raw)
#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)
#define fy_std_buffer_drop_v0 test_actual_buffer_drop
#define fy_private_vector_release_empty_v0 test_actual_vector_release
#define fy_std_file_open_v0 test_actual_file_open
#define fy_std_file_open_utf8_v0 test_actual_file_open_utf8
#define fy_std_file_close_v0 test_actual_file_close
#define fy_std_file_drop_v0 test_actual_file_drop
"""
    suffix = rb"""
#undef fy_std_buffer_drop_v0
#undef fy_private_vector_release_empty_v0
#undef fy_std_file_open_v0
#undef fy_std_file_open_utf8_v0
#undef fy_std_file_close_v0
#undef fy_std_file_drop_v0
uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *b){uint32_t s=test_actual_buffer_drop(b);++buffer_drops;if(s)++errors;return s;}
uint32_t fy_private_vector_release_empty_v0(void *p){uint32_t s=test_actual_vector_release(p);++vector_releases;if(s)++errors;return s;}
struct fy_std_windows_file_result_v0 fy_std_file_open_v0(const uint8_t *p,const struct fy_std_windows_open_options_v0 *o,void *s){struct fy_std_windows_file_result_v0 r=test_actual_file_open(p,o,s);if(!r.status)++file_opened;return r;}
struct fy_std_windows_file_result_v0 fy_std_file_open_utf8_v0(const uint8_t *p,size_t n,const struct fy_std_windows_open_options_v0 *o,void *s){struct fy_std_windows_file_result_v0 r=test_actual_file_open_utf8(p,n,o,s);if(!r.status)++file_opened;return r;}
struct fy_std_windows_file_result_v0 fy_std_file_close_v0(void *s){struct fy_std_windows_file_result_v0 r=test_actual_file_close(s);if(!r.status)++file_closed;return r;}
uint32_t fy_std_file_drop_v0(void *s){struct fy_std_windows_file_v0 before,after;int valid=fy_std_windows_file_load_v0(s,&before);uint32_t status=test_actual_file_drop(s);++file_dropped;if(status)++errors;if(valid&&before.open){++file_auto_closed;if(!fy_std_windows_file_load_v0(s,&after)||after.open||after.handle)++errors;if(GetHandleInformation((HANDLE)before.handle,&(DWORD){0})||GetLastError()!=ERROR_INVALID_HANDLE)++errors;}return status;}
_Static_assert(sizeof(struct fy_std_windows_file_v0)==16&&_Alignof(struct fy_std_windows_file_v0)==8&&offsetof(struct fy_std_windows_file_v0,open)==8,"runtime File layout");
extern int32_t fy_candidate_entry(void);
int main(void){int32_t status=fy_candidate_entry();const char *path=getenv("FY_TEST_RECEIPT");FILE *file=path?fopen(path,"wb"):NULL;if(!file)return 91;fprintf(file,"{\"heapAttempts\":%zu,\"heapLive\":%zu,\"bufferDrops\":%u,\"vectorReleases\":%u,\"cleanupErrors\":%u,\"fileOpened\":%u,\"fileClosed\":%u,\"fileDrops\":%u,\"fileAutomaticClose\":%u}\n",attempts,live,buffer_drops,vector_releases,errors,file_opened,file_closed,file_dropped,file_auto_closed);if(fclose(file))return 92;return status;}
"""
    runtime.write_bytes(prefix + b"\n".join(path.read_bytes() for path in components) + suffix)
    trap = output / "trap.c"
    trap.write_text("#include <stdint.h>\n#include <stdlib.h>\n_Noreturn void fy_runtime_trap(uint32_t category){exit(100+(int)category); }\n", encoding="utf-8")
    executions = []
    builds = []
    reference_runs = []
    with tempfile.TemporaryDirectory(prefix="file-token-inputs-", dir=output) as temporary:
        folder = Path(temporary)
        data = fixtures(folder)
        fixture_hashes = {name: hashlib.sha256(raw).hexdigest() for name, raw in data.items()}
        cases = [(name, [name]) for name in data] + [("missing", ["missing.fy"]), ("no-argument", []), ("extra-argument", ["empty.fy", "extra"])]
        expected = {}
        for name, argv in cases:
            run = binary_run(args.reference.resolve(), argv, folder, environment)
            reference_runs.append(dict(name=name, run=run))
            expected[name] = run
        assert all(expected[name]["exit"] == 0 for name in ("empty.fy", "source-한글-λ.fy", "literals.fy", "comments.fy", "exact.fy"))
        assert expected["overflow.fy"]["exit"] == expected["missing"]["exit"] == 2
        assert expected["no-argument"]["exit"] == expected["extra-argument"]["exit"] == 8
        counts_by_profile = {}
        for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
            binary = output / f"file-tokens-{profile}.exe"
            build = driver.run([str(compiler), "/nologo", "/std:c11", "/TC", flag,
                                "/DFY_CANDIDATE_NO_MAIN", str(generated), str(runtime), str(trap),
                                f"/Fe:{binary}", "/link", "/INCREMENTAL:NO"], output, environment)
            assert build["exit"] == 0, build
            builds.append(dict(profile=profile, build=build))
            counts_by_profile[profile] = {}
            for name, argv in cases:
                receipt_path = output / "oracle.json"
                receipt_path.unlink(missing_ok=True)
                env = dict(environment, FY_TEST_RECEIPT=str(receipt_path))
                run = binary_run(binary, argv, folder, env)
                oracle = json.loads(receipt_path.read_bytes())
                ref = expected[name]
                assert (run["exit"], run["stdout"], run["stderr"]) == (ref["exit"], ref["stdout"], ref["stderr"]), (name, run, ref)
                assert oracle["heapLive"] == oracle["cleanupErrors"] == 0, (name, oracle)
                assert oracle["fileOpened"] == oracle["fileClosed"] + oracle["fileAutomaticClose"], (name, oracle)
                counts_by_profile[profile][name] = oracle
                executions.append(dict(name=name, profile=profile, heapFailureAt=None, counts=oracle, run=run))
            attempts = counts_by_profile[profile]["literals.fy"]["heapAttempts"]
            assert attempts > 3
            for failure in range(1, attempts + 1):
                receipt_path = output / "oracle.json"
                receipt_path.unlink(missing_ok=True)
                env = dict(environment, FY_TEST_RECEIPT=str(receipt_path), FY_TEST_HEAP_FAIL_AT=str(failure))
                run = binary_run(binary, ["literals.fy"], folder, env)
                oracle = json.loads(receipt_path.read_bytes())
                assert run["exit"] == 2 and run["stdout"] == "" and run["stderr"] == "error: buffer allocation failed\n", (failure, run)
                assert oracle["heapAttempts"] == failure and oracle["heapLive"] == oracle["cleanupErrors"] == 0, (failure, oracle)
                assert oracle["fileOpened"] == oracle["fileClosed"] + oracle["fileAutomaticClose"]
                executions.append(dict(name="allocation-failure", profile=profile, heapFailureAt=failure, counts=oracle, run=run))
            observed = {path.name: hashlib.sha256(path.read_bytes()).hexdigest() for path in folder.iterdir()}
            assert observed == fixture_hashes, observed
        assert counts_by_profile["low"] == counts_by_profile["optimized"]
    receipt = dict(schema="fairyc.candidate-original-file-tokens-c-execution/v1", passed=True,
                   retainedInstances=int(proof[1]), sourceBytes=len(source), sourceSHA256=digest,
                   nativeReportSHA256=hashlib.sha256(args.report.read_bytes()).hexdigest(),
                   referenceSHA256=hashlib.sha256(args.reference.read_bytes()).hexdigest(),
                   builds=builds, executions=executions, referenceRuns=reference_runs,
                   fixtureSHA256=fixture_hashes, fixtureBytes={name:len(raw) for name,raw in data.items()},
                   runtimeInputs=[dict(path=p.relative_to(ROOT).as_posix(), sha256=hashlib.sha256(p.read_bytes()).hexdigest()) for p in components])
    (output / "receipt.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Original file tokens C passed: {len(reference_runs)} reference cases, {len(executions)} C runs")


if __name__ == "__main__":
    main()
