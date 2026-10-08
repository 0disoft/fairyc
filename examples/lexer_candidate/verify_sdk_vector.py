"""Observe Fairy-owned Vec compilation against the actual private C runtime."""
import argparse
import hashlib
import json
import os
import signal
from pathlib import Path
import subprocess

REPO = Path(__file__).resolve().parents[2]
SDK = REPO / "internal/sdkstdlib/sources/optional/std/vec.fy"
SUITE = "vector-storage"
TRAPS = {}
CASE_OBSERVER = Path(__file__)
FOREIGN = """
unsafe extern C fn usize fixture_live();
unsafe extern C fn void fixture_fail_next();
unsafe extern C fn usize fixture_trace_count();
unsafe extern C fn i32 fixture_trace_at(usize index);
"""
CASES = {
    "scalar-lifecycle": FOREIGN + """
fn i32 main(){
    {
        var v::Vec<i32> x=v::empty<i32>();
        assert(v::len<i32>(ref x)==0 && v::capacity<i32>(ref x)==0 && v::is_empty<i32>(ref x));
        assert(switch(v::pop<i32>(ref mut x)){case .none:{true}case .some(_):{false}});
        assert(v::reserve<i32>(ref mut x,0).is_ok);
        assert(v::reserve<i32>(ref mut x,3).is_ok);
        assert(v::push<i32>(ref mut x,11).is_ok && v::push<i32>(ref mut x,22).is_ok && v::push<i32>(ref mut x,33).is_ok);
        assert(v::len<i32>(ref x)==3 && !v::is_empty<i32>(ref x));
        assert(switch(v::pop<i32>(ref mut x)){case .some(n):{n==33}case .none:{false}});
        assert(v::push<i32>(ref mut x,44).is_ok);
        let usize capacity=v::capacity<i32>(ref x);
        assert(v::clear<i32>(ref mut x)==3 && v::capacity<i32>(ref x)==capacity);
        assert(v::push<i32>(ref mut x,55).is_ok);
        assert(v::reserve<i32>(ref mut x,capacity+1).is_ok);
        assert(unsafe{fixture_live()}==1);
    }
    assert(unsafe{fixture_live()}==0 && unsafe{fixture_trace_count()}==5);
    assert(unsafe{fixture_trace_at(0)}==33 && unsafe{fixture_trace_at(1)}==44 && unsafe{fixture_trace_at(2)}==22 && unsafe{fixture_trace_at(3)}==11 && unsafe{fixture_trace_at(4)}==55);
    return 7;
}
""",
    "oom-preserves-value": FOREIGN + """
fn i32 main(){
    {
        var v::Vec<i32> x=v::empty<i32>();unsafe{fixture_fail_next();}
        let Result<usize,i32> failed=v::push<i32>(ref mut x,-17);
        let i32 donor=switch(failed){case .err(n):{n}case .ok(_):{assert(false);0}};
        assert(donor==-17 && v::len<i32>(ref x)==0 && v::capacity<i32>(ref x)==0);
        assert(v::push<i32>(ref mut x,donor).is_ok);
        let usize capacity=v::capacity<i32>(ref x);unsafe{fixture_fail_next();}
        assert(!v::reserve<i32>(ref mut x,capacity+1).is_ok);
        assert(v::len<i32>(ref x)==1 && v::capacity<i32>(ref x)==capacity);
        assert(switch(v::pop<i32>(ref mut x)){case .some(n):{n==-17}case .none:{false}});
        var usize i=0;while(i<capacity){assert(v::push<i32>(ref mut x,9).is_ok);i+=1;}
        unsafe{fixture_fail_next();}
        assert(switch(v::push<i32>(ref mut x,42)){case .err(n):{n==42}case .ok(_):{false}});
        assert(v::len<i32>(ref x)==capacity && v::capacity<i32>(ref x)==capacity);
        assert(v::clear<i32>(ref mut x)==capacity);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "scalar-layouts": FOREIGN + """
fn i32 main(){
    {
        var v::Vec<u8> bytes=v::empty<u8>();assert(v::push<u8>(ref mut bytes,255).is_ok);
        assert(switch(v::pop<u8>(ref mut bytes)){case .some(n):{n==255}case .none:{false}});
        var v::Vec<i64> wide=v::empty<i64>();assert(v::push<i64>(ref mut wide,-9223372036854775808).is_ok);
        assert(switch(v::pop<i64>(ref mut wide)){case .some(n):{n==-9223372036854775808}case .none:{false}});
        var v::Vec<bool> flags=v::empty<bool>();assert(v::push<bool>(ref mut flags,true).is_ok);
        assert(switch(v::pop<bool>(ref mut flags)){case .some(n):{n}case .none:{false}});
        var v::Vec<u32> integers=v::empty<u32>();assert(v::push<u32>(ref mut integers,4294967295).is_ok);
        assert(switch(v::pop<u32>(ref mut integers)){case .some(n):{n==4294967295}case .none:{false}});
        var v::Vec<usize> sizes=v::empty<usize>();assert(v::push<usize>(ref mut sizes,18446744073709551615).is_ok);
        assert(switch(v::pop<usize>(ref mut sizes)){case .some(n):{n==18446744073709551615}case .none:{false}});
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
}
CASES["provider-first-fields"] = FOREIGN + """
struct Packet{i32 code;}
enum Kind{Some{i32 item;};None;}
fn Kind make(){return .Some{.item=9};}
fn i32 code(Kind k){return switch(k){case .Some{.item=t}:{t}case .None:{0}};}
fn i32 main(){
    let Packet packet=Packet{.code=9};var v::Vec<i32> x=v::empty<i32>();assert(v::push<i32>(ref mut x,packet.code+code(make())).is_ok);
    assert(switch(v::pop<i32>(ref mut x)){case .some(n):{n==18}case .none:{false}});
    return 7;
}
"""
REJECT = {
    "private-field": ('fn i32 main(){var v::Vec<i32> x=v::empty<i32>();discard x.length;return 7;}', "phase 5 code 6201"),
    "resource-element": ('resource Item{i32 n;cleanup{}}fn i32 main(){var v::Vec<Item> x=v::empty<Item>();return 7;}', "code 4502"),
    "view-access": ('fn i32 main(){var v::Vec<i32> x=v::empty<i32>();discard v::view<i32>(ref x);return 7;}', "phase 4 code 3"),
    "stolen-storage": ('fn i32 main(){discard unsafe{storage_size<i32>()};return 7;}', "phase 5 code 5102"),
    "stolen-runtime": ('@link_name("fy_private_vector_initialize_v0")unsafe extern C fn u32 stolen(ptr mut void p,usize size,usize alignment);fn i32 main(){return 7;}', "phase 1 code 5106"),
    "wrong-element": ('fn i32 main(){var v::Vec<i32> x=v::empty<i32>();discard v::push<i32>(ref mut x,true);return 7;}', "phase 5 code 1"),
    "immutable-borrow": ('fn i32 main(){let v::Vec<i32> x=v::empty<i32>();discard v::pop<i32>(ref mut x);return 7;}', "phase 5 code 4"),
}

PREFIX = r"""
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
static size_t fixture_live_blocks;
static int fixture_fail;
static int32_t fixture_trace[64];
static size_t fixture_trace_size;
static void *fixture_malloc(size_t size){
    if(fixture_fail){fixture_fail=0;return NULL;}
    void *p=malloc(size);if(p)fixture_live_blocks++;return p;
}
static void fixture_free(void *p){if(p){if(!fixture_live_blocks)abort();fixture_live_blocks--;free(p);}}
size_t fixture_live(void){return fixture_live_blocks;}
void fixture_fail_next(void){fixture_fail=1;}
size_t fixture_trace_count(void){return fixture_trace_size;}
int32_t fixture_trace_at(size_t i){if(i>=fixture_trace_size)abort();return fixture_trace[i];}
#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)
#define FY_STD_SYSTEM_HEAP_MALLOC_V0(size) fixture_malloc(size)
#define FY_STD_SYSTEM_HEAP_FREE_V0(data) fixture_free(data)
#define fy_private_vector_pop_slot_v0 fixture_pop_impl
"""
SUFFIX = r"""
#undef fy_private_vector_pop_slot_v0
void *fy_private_vector_pop_slot_v0(void *storage,size_t size){
    void *p=fixture_pop_impl(storage,size);
    if(p && size==sizeof(int32_t)){
        if(fixture_trace_size>=64)abort();
        memcpy(&fixture_trace[fixture_trace_size++],p,size);
    }
    return p;
}
"""


def identity(p):
    return dict(path=str(p), sha256=hashlib.sha256(p.read_bytes()).hexdigest())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--images", type=Path, nargs="+", required=True)
    ap.add_argument("--native-runner", type=Path, required=True)
    ap.add_argument("--receipt", type=Path, required=True)
    args = ap.parse_args()
    images = [p.resolve() for p in args.images]
    runner = args.native_runner.resolve()
    output = REPO / ".cache/candidate-tagged" / SUITE
    output.mkdir(parents=True, exist_ok=True)
    receipt = args.receipt.resolve()
    rows, jobs = [], []
    passed = False
    runtime_paths = [REPO / "internal/cstdlib" / p for p in ("stage0_system_heap.c", "stage0_vector_storage.c")]
    foreign = output / "runtime.c"
    foreign.write_text(PREFIX + "\n".join(p.read_text(encoding="utf-8") for p in runtime_paths) + SUFFIX, encoding="utf-8")

    def run(argv):
        p = subprocess.Popen([str(x) for x in argv], cwd=REPO, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                             encoding="utf-8", start_new_session=os.name != "nt")
        try:
            stdout, stderr = p.communicate(timeout=240)
        except subprocess.TimeoutExpired:
            if os.name == "nt":
                subprocess.run(["taskkill", "/PID", str(p.pid), "/T", "/F"], capture_output=True, timeout=30)
            else:
                os.killpg(p.pid, signal.SIGKILL)
            stdout, stderr = p.communicate(timeout=30)
            rows.append(dict(argv=[str(x) for x in argv], cwd=str(REPO), exit=p.returncode,
                             stdout=stdout, stderr=stderr, timedOut=True, timeoutSeconds=240))
            raise RuntimeError("Vec observer timed out")
        row = dict(argv=[str(x) for x in argv], cwd=str(REPO), exit=p.returncode, stdout=stdout, stderr=stderr)
        rows.append(row)
        return row

    def manifest(name, body):
        source = output / (name + ".fy")
        source.write_text("module main;import std::vec as v;" + body, encoding="utf-8")
        m = output / (name + ".txt")
        paths = [source.relative_to(REPO).as_posix(), SDK.relative_to(REPO).as_posix()]
        if name == "provider-first-fields":
            paths.reverse()
        m.write_text("\n".join(paths) + "\n", encoding="utf-8")
        return m

    native_report = output / "native.json"
    try:
        for name, body in CASES.items():
            m = manifest(name, body)
            for index, exe in enumerate(images):
                r = run([exe, "--c-modules-file", m])
                assert r["exit"] == 0 and not r["stderr"] and r["stdout"], r
                source = output / f"{name}-{index}.c"
                source.write_text(r["stdout"], encoding="utf-8", newline="\n")
                r["generatedC"] = identity(source)
                r["stdoutBytes"] = len(r["stdout"].encode("utf-8"))
                r["stdout"] = ""
                jobs.append(dict(name=f"{name}-{index}", source=str(source), observe=False, foreignSource=str(foreign)))
        for name, (body, diagnostic) in REJECT.items():
            m = manifest(name, body)
            for exe in images:
                r = run([exe, "--c-modules-file", m])
                assert r["exit"] == 2 and not r["stdout"] and diagnostic in r["stderr"], r
        request = output / "native-request.json"
        request.write_text(json.dumps(dict(schema="fairyc.candidate-c-native-request/v1", output=str(output / "native"), jobs=jobs)), encoding="utf-8")
        r = run([runner, request, native_report])
        assert r["exit"] == 0 and not r["stderr"], r
        native = json.loads(native_report.read_text(encoding="utf-8"))
        assert len(native) == 2 * len(jobs), native
        expected = {f"{name}-{index}": TRAPS.get(name, 0) for name in CASES for index in range(len(images))}
        assert all(not r.get("error") and r["trap"] == expected[r["name"]] and (r["trap"] != 0 or r["value"] == 7) and not r["stdout"] and not r["stderr"] for r in native), native
        passed = True
    finally:
        receipt.parent.mkdir(parents=True, exist_ok=True)
        receipt.write_text(json.dumps(dict(schema="fairyc.candidate-sdk-vector-native/v1", passed=passed,
            images=[identity(p) for p in images], runner=identity(runner), sdk=identity(SDK),
            runtime=[identity(p) for p in runtime_paths], observer=identity(Path(__file__)), caseObserver=identity(CASE_OBSERVER),
            sources=[identity(p) for p in sorted((REPO / "examples/lexer_candidate/src").glob("*.fy"))],
            lock=identity(REPO / "examples/lexer_candidate/fairy.lock"), executions=rows,
            native=json.loads(native_report.read_text(encoding="utf-8")) if passed else [],
            linuxValidated=False), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"PASS: {len(CASES)} Vec programs, {len(REJECT)} rejections, {len(native)} native runs")


if __name__ == "__main__":
    main()
