"""Execute private owned SDK C against the repository's real Windows runtime."""
import argparse
import base64
import hashlib
import importlib.util
import json
import os
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location("candidate_c", ROOT / "tools/verify-candidate-module-c.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)

# attempts, external SDK String/CString/Buffer drops, injected malloc failure.
TEXT_CASES = {
    "sdk_c_executes_string_utf8_views_append_and_empty": (2, 2, 0, 0, None),
    "sdk_c_executes_string_initial_allocation_failure": (1, 1, 0, 0, 1),
    "sdk_c_executes_string_append_failure_preserves_owner": (2, 1, 0, 0, 2),
    "sdk_c_executes_string_buffer_transfers_without_copy": (2, 2, 0, 3, None),
    "sdk_c_executes_cstring_termination_and_interior_nul": (2, 0, 3, 0, None),
    "sdk_c_executes_cstring_initial_allocation_failure": (1, 0, 1, 0, 1),
}

# Successful opens, explicit closes, cleanup calls, cleanup of open owners,
# heap attempts and injected failure. Programs run in isolated scratch folders.
FILE_CASES = {
    "sdk_c_executes_file_utf8_roundtrip_seek_and_close": (2, 2, 2, 0, 0, None),
    "sdk_c_executes_file_partial_eof_and_scope_cleanup": (2, 0, 2, 2, 0, None),
    "sdk_c_executes_file_open_error_and_exclusive_preservation": (2, 2, 5, 0, 0, None),
    "sdk_c_executes_file_replace_and_append": (4, 4, 5, 0, 0, None),
    "sdk_c_executes_file_buffer_read_to_end": (2, 2, 2, 0, 3, None),
    "sdk_c_executes_file_buffer_failure_preserves_cursor": (2, 2, 2, 0, 2, 2),
}
FILE_SOURCE_CASES = {
    "sdk_c_executes_original_file_loader_limits_and_errors": (4, 3, 6, 1, 4, None),
    "sdk_c_executes_original_file_loader_allocation_failure": (1, 0, 1, 1, 1, 1),
}
FILE_CASES.update(FILE_SOURCE_CASES)


def loader_fixtures(folder):
    data = {
        "small-λ.bin": b"A\0\xce\xbbZ",
        "empty.bin": b"",
        "exact.bin": bytes(index % 251 for index in range(65536)),
        "too-big.bin": bytes(index % 251 for index in range(65537)),
    }
    for name, raw in data.items():
        (folder / name).write_bytes(raw)
    return {name: hashlib.sha256(raw).hexdigest() for name, raw in data.items()}

CASES = {
    "byte_loop_c_evaluates_view_once_empty_and_nested_control": ([], [], ""),
    "byte_loop_c_preserves_iteration_defer_and_owner_cleanup": ([], [], "101,1,102,2,103,3,9,"),
    "sdk_c_executes_nested_vector_move_and_cleanup":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], "102,22,21,900,101,12,11,"),
    "sdk_c_executes_nested_vector_failed_push_donor":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], "107,72,71,900,"),
    "sdk_c_executes_vector_storage_and_views":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], ""),
    "sdk_c_executes_vector_resource_cleanup":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], "3,2,1,"),
    "sdk_c_executes_vector_failed_push_returns_donor":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], "7,"),
    "sdk_c_executes_vector_growth_failure_preserves_prefix":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], "9,8,7,6,5,4,3,2,1,"),
    "sdk_c_executes_vector_struct_slices":
        (["stage0_system_heap.c", "stage0_vector_storage.c"], [], ""),
    "sdk_c_executes_buffer_owned_views_and_cleanup":
        (["stage0_system_heap.c", "stage0_buffer.c"], [], ""),
    "sdk_c_executes_buffer_initial_allocation_failure":
        (["stage0_system_heap.c", "stage0_buffer.c"], [], ""),
    "sdk_c_executes_buffer_growth_failure_preserves_bytes":
        (["stage0_system_heap.c", "stage0_buffer.c"], [], ""),
    "pointer_c_executes_owned_raw_parts_and_zero_extent":
        ([], [], ""),
    "root_only_transitive_imports_keep_owned_identity":
        ([], [], ""),
    "sdk_c_executes_process_arguments_through_exact_result_bridge":
        (["stage0_windows_process_args.c"], ["abc", "λ"], ""),
    "sdk_c_executes_arena_factory_switch_and_owned_cleanup":
        (["stage0_system_heap.c", "stage0_arena.c"], [], ""),
    "sdk_c_executes_console_numbers_bytes_and_record_result_bridges":
        (["stage0_windows_console.c"], [], "42|-7|9|-9223372036854775808|λ\n"),
    "sdk_c_executes_stdin_and_stderr_byte_results":
        (["stage0_windows_console.c"], [], "A\0λZ"),
    "sdk_c_executes_region_lifecycle_and_cleanup_edges":
        (["stage0_system_heap.c", "stage0_arena.c"], [], "B,2,1,S,B,4,3,S,B,B,6,S,5,S,B,11,S,B,12,S,"),
    "sdk_c_executes_region_placement_and_param_origin":
        (["stage0_system_heap.c", "stage0_arena.c"], [], "B,7,S,"),
    "sdk_c_executes_region_storage_or_tracking_failure_as_result":
        (["stage0_system_heap.c", "stage0_arena.c"], [], "B,7,S,"),
}
CASES.update({name: (["stage0_system_heap.c", "stage0_buffer.c", "stage0_text_view.c", "stage0_owned_text.c"], [], "")
              for name in TEXT_CASES})
CASES.update({name: ((["stage0_system_heap.c", "stage0_buffer.c"] if "file_buffer_" in name else []) +
                    ["stage0_windows_file.c", "stage0_windows_file_portable.c"] +
                    (["stage0_windows_file_buffer.c"] if "file_buffer_" in name else []), [], "")
              for name in FILE_CASES})
CASES.update({name: (["stage0_system_heap.c", "stage0_buffer.c", "stage0_text_view.c",
                     "stage0_windows_file.c", "stage0_windows_file_portable.c"], [], "")
              for name in FILE_SOURCE_CASES})
PROOFS = {"byte_loop_c_rejects_forged_iteration_cfg","sdk_c_rejects_forged_cstring_view_authority","sdk_c_rejects_forged_storage_metadata","pointer_c_rejects_forged_raw_view_metadata","rejected_import_expansion_rolls_back_borrow_types","sdk_c_plan_respects_work_budget_before_authority_or_output",
          "sdk_c_region_plan_respects_placement_budget_before_output"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", type=Path, action="append", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--vcvars", type=Path, required=True)
    parser.add_argument("--case", choices=CASES, action="append")
    args = parser.parse_args()
    outcomes = {}
    reports = []
    for path in args.report:
        raw = path.read_bytes()
        report = json.loads(raw)
        assert report["engine"] == "native" and report["target"] == "x86_64-windows-msvc"
        for test in report["tests"]:
            prefix = next((prefix for prefix in ("tests::module_unit_imports::", "tests::module_unit_c_pointers::", "tests::module_unit_c_sdk::", "tests::module_unit_c_text::", "tests::module_unit_c_fs::", "tests::module_unit_c_file_source::", "tests::module_unit_c_iteration::") if test["identity"].startswith(prefix)), "")
            if prefix:
                name = test["identity"].removeprefix(prefix)
                if name in CASES or name in PROOFS:
                    outcomes[name] = test
        reports.append(dict(path=str(path), sha256=hashlib.sha256(raw).hexdigest()))
    required = set(args.case) if args.case else set(CASES) | PROOFS
    assert required <= set(outcomes), required - set(outcomes)
    failed = [(name, test["outcome"], test.get("trapCategory"))
              for name, test in outcomes.items() if name in required and test["outcome"] != "passed"]
    assert not failed, failed
    assert all(not base64.b64decode(outcomes[name]["stdoutBase64"], validate=True) for name in PROOFS & set(outcomes))
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
    args.output.mkdir(parents=True, exist_ok=True)
    output = args.output.resolve()
    trap = output / "trap.c"
    trap.write_text("#include <stdint.h>\n#include <stdlib.h>\n"
                    "_Noreturn void fy_runtime_trap(uint32_t category){exit(100+(int)category);}\n",
                    encoding="utf-8")
    executions = []
    for name in dict.fromkeys(args.case or CASES):
        components, argv, expected = CASES[name]
        source = base64.b64decode(outcomes[name]["stdoutBase64"], validate=True)
        assert source.startswith(b"/* fairyc.candidate-module-owned-c/v1 */\n")
        assert not base64.b64decode(outcomes[name]["stderrBase64"], validate=True)
        generated = output / f"{name}.c"
        generated.write_bytes(source)
        inputs = [ROOT / "internal/cstdlib" / component for component in components]
        # Arena shares the existing SystemHeap component and its status/type
        # definitions in one translation unit, as the normal runtime builder does.
        runtime = output / f"{name}-runtime.c"
        arena = "stage0_arena.c" in components
        buffered = "stage0_buffer.c" in components
        vector = "stage0_vector_storage.c" in components
        textual = name in TEXT_CASES
        filing = name in FILE_CASES
        loading = name in FILE_SOURCE_CASES
        heap_checked = buffered or vector or name in {"sdk_c_executes_region_placement_and_param_origin", "sdk_c_executes_region_storage_or_tracking_failure_as_result"}
        placing = name in {"sdk_c_executes_region_placement_and_param_origin",
                           "sdk_c_executes_region_storage_or_tracking_failure_as_result"}
        regions = name == "sdk_c_executes_region_lifecycle_and_cleanup_edges" or placing
        arena_count = 1 if placing else 2
        region_count = 1 if placing else 6
        heap = b""
        if heap_checked:
            heap = (
                b"#include <stddef.h>\n#include <stdlib.h>\n"
                b"static size_t heap_attempts,heap_live;\n"
                b"static void *test_heap_malloc(size_t size){++heap_attempts;"
                b"const char *fail=getenv(\"FY_TEST_HEAP_FAIL_AT\");"
                b"if(fail&&strtoul(fail,NULL,10)==heap_attempts)return NULL;"
                b"void *raw=malloc(size);if(raw)++heap_live;return raw;}\n"
                b"static void test_heap_free(void *raw){if(raw){if(!heap_live)abort();--heap_live;}free(raw);}\n"
                b"#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) test_heap_malloc(size)\n"
                b"#define FY_STD_SYSTEM_HEAP_FREE_V0(raw) test_heap_free(raw)\n")
        instrumentation = b""
        if name == "byte_loop_c_preserves_iteration_defer_and_owner_cleanup":
            instrumentation = (b"#include <stdint.h>\n#include <stdio.h>\n"
                               b"void test_iteration_report(int32_t n){printf(\"%d,\",(int)n);}\n")
        if arena:
            instrumentation = (
                b"#undef fy_std_arena_initialize_v0\n#undef fy_std_arena_drop_v0\n"
                b"static unsigned initialized,dropped,invalid_status;\n"
                b"uint32_t fy_std_arena_initialize_v0(struct fy_std_arena_v0 *arena){"
                b"uint32_t status=fy_test_actual_arena_initialize_v0(arena);"
                b"++initialized;if(status)++invalid_status;return status;}\n"
                b"uint32_t fy_std_arena_drop_v0(struct fy_std_arena_v0 *arena){"
                b"uint32_t status=fy_test_actual_arena_drop_v0(arena);"
                b"++dropped;if(status)++invalid_status;return status;}\n"
                b"extern int32_t fy_candidate_entry(void);\n"
                b"int main(void){int32_t result=fy_candidate_entry();"
                b"return result?result:(initialized==2&&dropped==2&&!invalid_status?0:90);}\n")
            instrumentation = instrumentation.replace(b"initialized==2&&dropped==2", f"initialized=={arena_count}&&dropped=={arena_count}".encode())
            if regions:
                instrumentation = instrumentation.replace(
                    b"extern int32_t fy_candidate_entry(void);", (
                        b"#undef fy_std_region_begin_v0\n#undef fy_std_region_reset_v0\n"
                        b"#include <stdio.h>\nstatic unsigned begun,resets;\n"
                        b"void test_report(int32_t n){printf(\"%d,\",(int)n);}\n"
                        b"uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 *a,struct fy_std_region_v0 *r){"
                        b"uint32_t status=fy_test_actual_region_begin_v0(a,r);++begun;"
                        b"if(status)++invalid_status;printf(\"B,\");return status;}\n"
                        b"uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 *r){"
                        b"uint32_t status=fy_test_actual_region_reset_v0(r);++resets;"
                        b"if(status)++invalid_status;printf(\"S,\");return status;}\n"
                        b"extern int32_t fy_candidate_entry(void);"))
                instrumentation = instrumentation.replace(b"&&!invalid_status?", f"&&begun=={region_count}&&resets=={region_count}&&!invalid_status?".encode())
            if placing:
                attempts = (b'strtoul(getenv("FY_TEST_HEAP_FAIL_AT"),NULL,10)'
                            if name == "sdk_c_executes_region_storage_or_tracking_failure_as_result" else b"10")
                instrumentation = instrumentation.replace(b"&&!invalid_status?", b"&&heap_live==0&&heap_attempts==" + attempts + b"&&!invalid_status?")
        buffer_drops = 2 if name == "sdk_c_executes_buffer_owned_views_and_cleanup" else 1
        buffer_failure = (1 if name == "sdk_c_executes_buffer_initial_allocation_failure" else
                          2 if name == "sdk_c_executes_buffer_growth_failure_preserves_bytes" else None)
        if buffered:
            instrumentation = (
                b"#undef fy_std_buffer_drop_v0\n#include <stdint.h>\n"
                b"static unsigned buffer_dropped,buffer_drop_errors;\n"
                b"uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *b){"
                b"uint32_t status=fy_test_actual_buffer_drop_v0(b);++buffer_dropped;"
                b"if(status)++buffer_drop_errors;return status;}\n"
                b"extern int32_t fy_candidate_entry(void);\nint main(void){int32_t result=fy_candidate_entry();"
                + f"return result?result:(buffer_dropped=={buffer_drops}&&!buffer_drop_errors&&heap_live==0&&heap_attempts=={buffer_failure or 3}?0:90);}}\n".encode())
        vector_failure = (1 if name in {"sdk_c_executes_vector_failed_push_returns_donor",
                                       "sdk_c_executes_nested_vector_failed_push_donor"} else
                          2 if name == "sdk_c_executes_vector_growth_failure_preserves_prefix" else None)
        vector_attempts = 2 if name in {"sdk_c_executes_vector_storage_and_views",
                                       "sdk_c_executes_vector_growth_failure_preserves_prefix"} else 1
        if vector:
            instrumentation = (
                b"#undef fy_private_vector_release_empty_v0\n#include <stdio.h>\n"
                b"static unsigned vector_releases,vector_release_errors;\n"
                b"void test_report(int32_t n){printf(\"%d,\",(int)n);}\n"
                b"uint32_t fy_private_vector_release_empty_v0(void *owner){"
                b"uint32_t status=fy_test_actual_vector_release_empty_v0(owner);++vector_releases;"
                b"if(status)++vector_release_errors;return status;}\n"
                b"extern int32_t fy_candidate_entry(void);\nint main(void){int32_t result=fy_candidate_entry();"
                + f"return result?result:(vector_releases==1&&!vector_release_errors&&heap_live==0&&heap_attempts=={vector_attempts}?0:90);}}\n".encode())
        if textual:
            attempts, string_drops, cstring_drops, buffer_drops, text_failure = TEXT_CASES[name]
            instrumentation = (
                b"#undef fy_std_string_drop_storage_v0\n#undef fy_std_cstring_drop_storage_v0\n#undef fy_std_buffer_drop_v0\n"
                b"static unsigned string_dropped,cstring_dropped,buffer_dropped,text_drop_errors;\n"
                b"struct fy_std_owned_text_result_v0 fy_std_string_drop_storage_v0(void *owner){"
                b"struct fy_std_owned_text_result_v0 result=fy_test_actual_string_drop_storage_v0(owner);"
                b"++string_dropped;if(result.status||result.allocation_status)++text_drop_errors;return result;}\n"
                b"struct fy_std_owned_text_result_v0 fy_std_cstring_drop_storage_v0(void *owner){"
                b"struct fy_std_owned_text_result_v0 result=fy_test_actual_cstring_drop_storage_v0(owner);"
                b"++cstring_dropped;if(result.status||result.allocation_status)++text_drop_errors;return result;}\n"
                b"uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *owner){uint32_t status=fy_test_actual_buffer_drop_v0(owner);"
                b"++buffer_dropped;if(status)++text_drop_errors;return status;}\n"
                b"int32_t test_cstring_probe(uint8_t const *data,size_t length){if(!data)return 0;"
                b"return length==0?data[0]==0:length==3&&data[0]==206&&data[1]==187&&data[2]==90&&data[3]==0;}\n"
                b"_Static_assert(sizeof(struct fy_std_string_v0)==32&&_Alignof(struct fy_std_string_v0)==8,\"runtime String storage\");\n"
                b"_Static_assert(sizeof(struct fy_std_cstring_v0)==32&&_Alignof(struct fy_std_cstring_v0)==8,\"runtime CString storage\");\n"
                b"extern int32_t fy_candidate_entry(void);\nint main(void){int32_t result=fy_candidate_entry();"
                + f"return result?result:(string_dropped=={string_drops}&&cstring_dropped=={cstring_drops}&&buffer_dropped=={buffer_drops}&&!text_drop_errors&&heap_live==0&&heap_attempts=={attempts}?0:90);}}\n".encode())
        if filing:
            opened, closed, drops, auto_closed, attempts, file_failure = FILE_CASES[name]
            file_rename = (
                b"#define fy_std_file_open_v0 fy_test_actual_file_open_v0\n"
                b"#define fy_std_file_open_utf8_v0 fy_test_actual_file_open_utf8_v0\n"
                b"#define fy_std_file_close_v0 fy_test_actual_file_close_v0\n"
                b"#define fy_std_file_drop_v0 fy_test_actual_file_drop_v0\n")
            instrumentation = (
                b"#undef fy_std_file_open_v0\n#undef fy_std_file_open_utf8_v0\n"
                b"#undef fy_std_file_close_v0\n#undef fy_std_file_drop_v0\n"
                b"static unsigned file_opened,file_closed,file_dropped,file_auto_closed,file_drop_errors;\n"
                b"struct fy_std_windows_file_result_v0 fy_std_file_open_v0(const uint8_t *p,const struct fy_std_windows_open_options_v0 *o,void *s){"
                b"struct fy_std_windows_file_result_v0 r=fy_test_actual_file_open_v0(p,o,s);if(!r.status)++file_opened;return r;}\n"
                b"struct fy_std_windows_file_result_v0 fy_std_file_open_utf8_v0(const uint8_t *p,size_t n,const struct fy_std_windows_open_options_v0 *o,void *s){"
                b"struct fy_std_windows_file_result_v0 r=fy_test_actual_file_open_utf8_v0(p,n,o,s);if(!r.status)++file_opened;return r;}\n"
                b"struct fy_std_windows_file_result_v0 fy_std_file_close_v0(void *s){"
                b"struct fy_std_windows_file_result_v0 r=fy_test_actual_file_close_v0(s);if(!r.status)++file_closed;return r;}\n"
                b"uint32_t fy_std_file_drop_v0(void *s){struct fy_std_windows_file_v0 before,after;"
                b"int valid=fy_std_windows_file_load_v0(s,&before);uint32_t status=fy_test_actual_file_drop_v0(s);++file_dropped;"
                b"if(status)++file_drop_errors;if(valid&&before.open){++file_auto_closed;"
                b"if(!fy_std_windows_file_load_v0(s,&after)||after.open||after.handle)++file_drop_errors;"
                b"if(GetHandleInformation((HANDLE)before.handle,&(DWORD){0})||GetLastError()!=ERROR_INVALID_HANDLE)++file_drop_errors;}return status;}\n"
                b"_Static_assert(sizeof(struct fy_std_windows_file_v0)==16&&_Alignof(struct fy_std_windows_file_v0)==8&&offsetof(struct fy_std_windows_file_v0,open)==8,\"runtime File layout\");\n")
            if buffered:
                buffer_drops = 4 if name == "sdk_c_executes_original_file_loader_limits_and_errors" else 1
                instrumentation += (
                    b"#undef fy_std_buffer_drop_v0\nstatic unsigned buffer_dropped,buffer_drop_errors;\n"
                    b"uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 *b){uint32_t r=fy_test_actual_buffer_drop_v0(b);"
                    b"++buffer_dropped;if(r)++buffer_drop_errors;return r;}\n")
            checks = f"file_opened=={opened}&&file_closed=={closed}&&file_dropped=={drops}&&file_auto_closed=={auto_closed}&&!file_drop_errors"
            if buffered:
                checks += f"&&heap_live==0&&heap_attempts=={attempts}&&buffer_dropped=={buffer_drops}&&!buffer_drop_errors"
            instrumentation += ("extern int32_t fy_candidate_entry(void);\nint main(void){int32_t result=fy_candidate_entry();"
                                f"return result?result:({checks}?0:90);}}\n").encode()
        runtime.write_bytes(heap + b"#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)\n" +
                            (file_rename if filing else b"") +
                            (b"#define fy_std_string_drop_storage_v0 fy_test_actual_string_drop_storage_v0\n"
                             b"#define fy_std_cstring_drop_storage_v0 fy_test_actual_cstring_drop_storage_v0\n" if textual else b"") +
                            (b"#define fy_private_vector_release_empty_v0 fy_test_actual_vector_release_empty_v0\n" if vector else b"") +
                            (b"#define fy_std_buffer_drop_v0 fy_test_actual_buffer_drop_v0\n" if buffered else b"") +
                            (b"#define fy_std_arena_initialize_v0 fy_test_actual_arena_initialize_v0\n"
                             b"#define fy_std_arena_drop_v0 fy_test_actual_arena_drop_v0\n" if arena else b"") +
                            (b"#define fy_std_region_begin_v0 fy_test_actual_region_begin_v0\n"
                             b"#define fy_std_region_reset_v0 fy_test_actual_region_reset_v0\n" if regions else b"") +
                            b"\n".join(path.read_bytes() for path in inputs) + b"\n" + instrumentation)
        for profile, flag in (("low", "/Od"), ("optimized", "/O2")):
            binary = output / f"{name}-{profile}.exe"
            build = driver.run([str(compiler), "/nologo", "/std:c11", "/TC", flag,
                                *(["/DFY_CANDIDATE_NO_MAIN"] if arena or buffered or vector or filing else []),
                                str(generated), str(runtime), str(trap), f"/Fe:{binary}",
                                "/link", "/INCREMENTAL:NO"], output, environment)
            assert build["exit"] == 0, build
            reading = name == "sdk_c_executes_stdin_and_stderr_byte_results"
            stdin = b"A\0\xce\xbbZ" if reading else None
            expected_stderr = "42|-7|9|-9223372036854775808|true|λ\n" if reading else ""
            failures = (file_failure,) if filing else (text_failure,) if textual else (1, 2) if name == "sdk_c_executes_region_storage_or_tracking_failure_as_result" else (buffer_failure,) if buffered else (vector_failure,) if vector else (None,)
            for failure in failures:
                run_environment = dict(os.environ)
                run_environment.pop("FY_TEST_HEAP_FAIL_AT", None)
                if failure is not None:
                    run_environment["FY_TEST_HEAP_FAIL_AT"] = str(failure)
                if filing:
                    with tempfile.TemporaryDirectory(prefix="candidate-file-", dir=output) as scratch:
                        fixture_hashes = loader_fixtures(Path(scratch)) if loading else None
                        run = driver.run([str(binary)] + argv, Path(scratch), run_environment, stdin=stdin)
                        assert run["exit"] == 0 and run["stdout"] == expected and run["stderr"] == expected_stderr, run
                        if loading:
                            observed = {path.name: hashlib.sha256(path.read_bytes()).hexdigest()
                                        for path in Path(scratch).iterdir()}
                            assert observed == fixture_hashes, (name, observed)
                        else:
                            assert not list(Path(scratch).iterdir()), f"File fixture left files behind: {name}"
                else:
                    run = driver.run([str(binary)] + argv, output, run_environment, stdin=stdin)
                assert run["exit"] == 0 and run["stdout"] == expected and run["stderr"] == expected_stderr, run
                executions.append(dict(name=name, profile=profile, sourceSHA256=hashlib.sha256(source).hexdigest(),
                                   stdinBase64=base64.b64encode(stdin).decode('ascii') if stdin is not None else None,
                                   heapFailureAt=failure, heapBalanceVerified=heap_checked,
                                   heapCounts={"attempts": failure or (attempts if textual or filing else vector_attempts if vector else 3 if buffered else 10), "liveAfter": 0} if heap_checked else None,
                                   textCounts={"stringDrop": string_drops, "cstringDrop": cstring_drops, "dropStatusErrors": 0} if textual else None,
                                   fileCounts={"open": opened, "close": closed, "drop": drops, "automaticClose": auto_closed,
                                               "dropStatusErrors": 0, "closedHandleVerified": auto_closed > 0} if filing else None,
                                   loaderFixtures=fixture_hashes if loading else None,
                                   vectorCounts={"release": 1, "releaseStatusErrors": 0} if vector else None,
                                   bufferCounts={"drop": buffer_drops, "dropStatusErrors": 0} if buffered else None,
                                   arenaCounts={"initialize": arena_count, "drop": arena_count, "nonzeroStatus": 0} if arena else None,
                                   regionCounts={"begin": region_count, "reset": region_count, "nonzeroStatus": 0} if regions else None,
                                   runtimeInputs=[dict(path=str(path.relative_to(ROOT)),
                                                      sha256=hashlib.sha256(path.read_bytes()).hexdigest())
                                                  for path in inputs], build=build, run=run))
    receipt = dict(schema="fairyc.candidate-sdk-c-execution/v1", passed=True,
                   reports=reports, executions=executions, compiler=str(compiler),
                   source="Private Fairy emitter; actual repository C runtime components, no SDK stand-ins")
    (output / "receipt.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"SDK C execution passed: {len(executions)} runs")


if __name__ == "__main__":
    main()
