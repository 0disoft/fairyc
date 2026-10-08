"""Observe native Fairy module API tests; does not implement compiler logic."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import os

PROJECT = Path(__file__).resolve().parent
ROOT = PROJECT.parents[1]

CLI_CASES = (
    ("cstr-import", 'module main;import provider as p;fn i32 main(){let ref CStr s=p::identity(c"AB");let ptr u8 data=unsafe{s as ptr u8};assert(data!=null);return 7;}',
     "module provider;pub fn ref CStr identity(ref CStr s){return s;}", "", 7),
    ("struct", "module main;import provider as p;struct Token{i32 n;}fn i32 main(){let p::Token t=p::make();return p::value(t);}",
     "module provider;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}pub fn i32 value(Token t){return t.n;}", "", 7),
    ("helpers", "module main;import provider as p;fn i32 helper(){return 9;}fn i32 main(){return p::value();}",
     "module provider;fn i32 helper(){return 7;}pub fn i32 value(){return helper();}", "", 7),
    ("resource", "module main;import provider as p;@effects(console)fn i32 main(){let p::Owner x=p::make();return p::value(ref x);}",
     'module provider;@effects(console)@link_name("fixture_drop")unsafe extern C fn void dropped(i32 n);@effects(console)pub resource Owner{i32 n;cleanup{unsafe{dropped(this.n);}}}pub fn Owner make(){return Owner{.n=65};}pub fn i32 value(ref Owner x){return x.n-58;}', "A", 7),
    ("enum", "module main;import provider as p;import provider as q;fn i32 main(){let p::Kind k=q::make();return p::value(k);}",
     "module provider;pub enum Kind{Off;On;}pub fn Kind make(){return .On;}pub fn i32 value(Kind k){return switch(k){case .Off:{0}case .On:{7}};}", "", 7),
    ("repr-provider", "module main;import provider as p;fn i32 main(){let p::Packet t=p::make();assert(t.code==41 && t.amount==1);return 7;}",
     "module provider;@repr(C)pub struct Packet{u32 code;u32 amount;}unsafe extern C fn Packet fixture_packet();pub fn Packet make(){return unsafe{fixture_packet()};}", "", 7),
    ("repr-import-result", "module main;import provider as p;unsafe extern C fn p::Packet fixture_packet();fn i32 main(){let p::Packet t=unsafe{fixture_packet()};return p::value(ref t);}",
     "module provider;@repr(C)pub struct Packet{u32 code;u32 amount;}pub fn i32 value(ref Packet p){assert(p.code==41 && p.amount==1);return 7;}", "", 7),
    ("branch-stack", "module main;struct Large{u32 a;u32 b;u32 c;u32 d;u32 e;}fn i32 main(){let Large p=Large{.a=1,.b=2,.c=3,.d=4,.e=5};assert(p.a==1 && p.b==2 && p.c==3 && p.d==4 && p.e==5);return 7;}",
     "module provider;", "", 7),
    ("raw-import", "module main;import provider as p;fn i32 main(){let ref u8[] owner=b\"AB\";let ptr u8 data=unsafe{owner as ptr u8};let ref u8[] v=p::wrap(owner,data,2);assert(v[1]==66);return 7;}",
     "module provider;pub fn ref u8[] wrap(ref u8[] owner,ptr u8 data,usize length){return unsafe{byte_slice::from_raw_parts(owner,data,length)};}", "", 7),
)


def verify_cli(images, runner, compiler, bootstrap_receipt, receipt=None, lane="modules"):
    generics = lane == "generics"
    sdk_num = lane == "num"
    sdk_process = lane == "process"
    cases = (
        ("generic-provider", "module main;import provider as p;import provider as q;fn T id<T>(T x){return x;}fn i32 main(){assert(p::id<bool>(true));return p::id<i32>(q::id<i32>(id<i32>(7)));}",
         "module provider;fn T helper<T>(T x){return x;}pub fn T id<T>(T x){return helper<T>(x);}", "", 7),
        ("generic-nominal", "module main;import provider as p;fn i32 main(){return p::pick<i32>(7,p::make());}",
         "module provider;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}pub fn T pick<T>(T x,Token t){assert(t.n==7);return x;}", "", 7),
    ) if generics else CLI_CASES
    sdk_source = ROOT / ("internal/sdkstdlib/sources/std/process.fy" if sdk_process else "internal/sdkstdlib/sources/std/num.fy")
    if sdk_process:
        sdk = sdk_source.read_text(encoding="utf-8")
        cases = (
            ("sdk-process-count", 'module main;import std::process as p;fn i32 main(){assert(p::arg_count()>0);return 7;}', sdk, "", 7),
            ("borrow-before-if-without-else", 'module main;fn void take(ref mut u8[] value){}fn i32 main(){var u8[1] a=[0];let ref mut u8[] parent=ref mut a;take(parent[..0]);if(true){var u8[1] c=[0];take(ref mut c);}return 7;}', "module provider;", "", 7),
            ("sdk-process-arg", 'module main;import std::process as p;fn i32 main(){var u8[4096] data=[65;4096];var u8[1] empty=[65];let ref mut u8[] small=ref mut empty;let usize needed=switch(p::arg(0,small[..0])){case .ok(_):{assert(false);0}case .err(e):{assert(e.status==3 && e.needed>0);e.needed}};assert(empty[0]==65 && data[0]==65);let usize n=switch(p::arg(0,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(n==needed && data[0]!=0);if(p::arg_count()==3){let usize n=switch(p::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] expected=b"한글";assert(n==expected.len);var usize i=0;while(i<n){assert(data[i]==expected[i]);i+=1;}assert(switch(p::arg(2,ref mut data)){case .ok(n):{n==0}case .err(_):{false}});}assert(switch(p::arg(p::arg_count(),ref mut data)){case .ok(_):{false}case .err(e):{e.status==1 && e.needed==0}});return 7;}', sdk, "", 7),
        )
    if sdk_num:
        sdk = sdk_source.read_text(encoding="utf-8")
        cases = (
            ("sdk-num-parse", 'module main;import std::num as n;fn i64 parse(ref u8[] s){return switch(n::parse_i64_decimal(s)){case .ok(v):{v}case .err(_):{assert(false);0}};}fn i32 error(ref u8[] s){return switch(n::parse_i64_decimal(s)){case .ok(_):{0}case .err(e):{switch(e){case .Empty:{1}case .InvalidDecimal:{2}case .OutOfRange:{3}}}};}fn i32 main(){assert(parse(b"0")==0 && parse(b"-0")==0 && parse(b"4000000000")==4000000000);assert(parse(b"9223372036854775807")==9223372036854775807 && parse(b"-9223372036854775808")==-9223372036854775808);assert(error(b"")==1 && error(b"-")==2 && error(b"+1")==2 && error(b"1x")==2);assert(error(b"9223372036854775808")==3 && error(b"-9223372036854775809")==3);return 7;}', sdk, "", 7),
            ("sdk-num-convert", 'module main;import std::num as n;fn i32 main(){assert(switch(n::u8_from_i64(255)){case .ok(v):{v==255}case .err(_):{false}});assert(switch(n::u8_from_i64(-1)){case .ok(_):{false}case .err(e):{switch(e){case .Negative:{true}case .OutOfRange:{false}}}});assert(!n::u8_from_i64(256).is_ok);assert(switch(n::i64_from_usize(9223372036854775807)){case .ok(v):{v==9223372036854775807}case .err(_):{false}});assert(!n::i64_from_usize(18446744073709551615).is_ok);assert(switch(n::usize_from_i64(9223372036854775807)){case .ok(v):{v==9223372036854775807}case .err(_):{false}});assert(!n::usize_from_i64(-1).is_ok);return 7;}', sdk, "", 7),
            ("enum-local", 'module main;enum Error{Bad;Good;}struct Item{Option<Error> value;}fn Result<i64,Error> make(bool ok){return if(ok){.ok(4000000000)}else{.err(.Bad)};}fn i32 main(){let Item x=Item{.value=.some(.Good)};assert(switch(x.value){case .some(e):{switch(e){case .Bad:{false}case .Good:{true}}}case .none:{false}});assert(switch(make(true)){case .ok(v):{v==4000000000}case .err(_):{false}});assert(switch(make(false)){case .ok(_):{false}case .err(e):{switch(e){case .Bad:{true}case .Good:{false}}}});return 7;}', "module provider;", "", 7),
        )
    output = ROOT / (".cache/candidate-sdk-process-cli" if sdk_process else ".cache/candidate-sdk-num-cli" if sdk_num else ".cache/candidate-module-generics-cli" if generics else ".cache/candidate-module-cli")
    output.mkdir(parents=True, exist_ok=True)
    rows, fixtures, jobs = [], [], []

    def digest(path):
        return hashlib.sha256(path.read_bytes()).hexdigest()

    def save(path, value):
        path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8", newline="\n")

    def run(argv, expected=0):
        result = subprocess.run(list(map(str, argv)), cwd=PROJECT, capture_output=True,
                                encoding="utf-8", timeout=360)
        row = dict(argv=list(map(str, argv)), cwd=str(PROJECT), exit=result.returncode,
                   stdout=result.stdout, stderr=result.stderr)
        rows.append(row)
        save(output / "raw.json", rows)
        assert result.returncode == expected, row
        return row

    def manifest(name, first, second):
        files = [output / (name + "-main.fy"), output / (name + "-provider.fy")]
        for path, source in zip(files, (first, second)):
            path.write_text(source, encoding="utf-8", newline="\n")
        path = output / (name + ".list")
        path.write_text("\n".join(os.path.relpath(f, PROJECT) for f in files) + "\n", encoding="utf-8", newline="\n")
        fixtures.extend(dict(path=str(f), sha256=digest(f)) for f in files + [path])
        return path

    run([compiler, "check", "--locked"])

    provider = output / "provider.c"
    provider.write_text('#include <stdint.h>\n#include <stdio.h>\nvoid fixture_drop(int32_t n){(void)fputc(n,stdout);}\nstruct Packet{uint32_t code;uint32_t amount;};\nstruct Packet fixture_packet(void){struct Packet p={41,1};return p;}\n', encoding="utf-8", newline="\n")
    runtime_sources = []
    if sdk_process:
        runtime_sources = [ROOT / ("internal/cstdlib/stage0_windows_process_args.c" if os.name == "nt" else "internal/cstdlib/stage0_linux_process_args.c")]
        if os.name != "nt":
            runtime_sources.insert(0, ROOT / "internal/cstdlib/stage0_text_view.c")
        provider.write_text("\n".join(p.read_text(encoding="utf-8") for p in runtime_sources), encoding="utf-8", newline="\n")
    for name, first, second, trace, value in cases:
        path = manifest(name, first, second)
        emitted = [run([image, "--c-modules-file", path]) for image in images]
        assert all(not row["stderr"] and row["stdout"] == emitted[0]["stdout"] for row in emitted)
        if name == "enum-local":
            for image in images:
                single = run([image, "--c-file", output / (name + "-main.fy")])
                assert single["stdout"] == emitted[0]["stdout"] and not single["stderr"]
        if name == "helpers":
            for image in images:
                prior = run([image, "--check-modules-file", path])
                assert prior["stdout"] == "modules ok\n" and not prior["stderr"], prior
        source = output / (name + ".observed.c")
        entry = ('void fy_std_process_set_args_v0(int,char**);\nint main(int argc,char **argv){fy_std_process_set_args_v0(argc,argv);' if sdk_process and os.name != "nt" else 'int main(void){')
        source.write_text('#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n' + emitted[0]["stdout"] + entry +
                          'int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t n=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",n);return 0;}\n', encoding="utf-8", newline="\n")
        jobs.append(dict(name=name, source=str(source), foreignSource=str(provider), observe=True))
    rejections = (
        ("generic-private", "module main;import provider as p;fn i32 main(){return p::id<i32>(7);}",
         "module provider;fn T id<T>(T x){return x;}", "phase 3 code 6"),
        ("generic-mismatch", "module main;import provider as p;fn i32 main(){discard p::id<bool>(7);return 0;}",
         "module provider;pub fn T id<T>(T x){return x;}", "phase 5 code 1"),
        ("generic-aggregate", "module main;fn i32 main(){return 0;}",
         "module provider;pub struct Box<T>{T x;}", "phase 2 code 4501"),
        ("generic-body-error", "module main;import provider as p;fn i32 main(){return p::id<i32>(7);}",
         "module provider;pub fn T id<T>(T x){return missing;}", "-provider.fy:1:"),
    ) if generics else (
        ("private", "module main;import provider as p;fn i32 main(){return p::secret();}",
         "module provider;fn i32 secret(){return 7;}", "phase 3 code 6"),
        ("field", "module main;import provider as p;fn i32 main(){let p::Owner x=p::make();return x.n;}",
         "module provider;pub resource Owner{i32 n;cleanup{}}pub fn Owner make(){return Owner{.n=7};}", "phase 5 code 6201"),
        ("provider-error", "module main;import provider as p;fn i32 main(){return p::value();}",
         "module provider;pub fn i32 value(){return missing;}", "-provider.fy:1:"),
        ("no-entry", "module main;fn i32 value(){return 7;}", "module provider;", "exactly one fn i32 main"),
        ("repr-import-plain", "module main;import provider as p;unsafe extern C fn p::Packet fixture_packet();fn i32 main(){return 7;}",
         "module provider;pub struct Packet{u32 code;u32 amount;}", "code 5102"),
        ("repr-private", "module main;import provider as p;unsafe extern C fn p::Packet fixture_packet();fn i32 main(){return 7;}",
         "module provider;@repr(C)struct Packet{u32 code;u32 amount;}", "code 6"),
    )
    if sdk_num:
        rejections = (
            ("sdk-tampered", "module main;import std::num as n;fn i32 main(){return 7;}", sdk + "\n", "phase 1 code 6801"),
            ("sdk-forged", "module main;import std::num as n;fn i32 main(){return 7;}", "module std::num;pub enum ParseI64Error{Empty;InvalidDecimal;OutOfRange;}", "phase 1 code 6801"),
            ("enum-distinct", 'module main;import std::num as n;enum ParseI64Error{Empty;InvalidDecimal;OutOfRange;}fn Result<i64,ParseI64Error> wrong(){return n::parse_i64_decimal(b"1");}fn i32 main(){return 7;}', sdk, "phase 5 code 1"),
        )
    if sdk_process:
        rejections = (
            ("sdk-process-tampered", "module main;fn i32 main(){return 7;}", sdk + "\n", "phase 1 code 6801"),
            ("sdk-process-forged", "module main;fn i32 main(){return 7;}", 'module std::process;@link_name("fy_std_process_arg_count_v0")unsafe extern C fn usize process_arg_count_raw();', "phase 1 code 6801"),
            ("sdk-process-stolen", 'module main;@link_name("fy_std_process_arg_count_v0")unsafe extern C fn usize stolen();fn i32 main(){return 7;}', sdk, "phase 1 code 5106"),
        )
    for name, first, second, diagnostic in rejections:
        path = manifest(name, first, second)
        for image in images:
            row = run([image, "--c-modules-file", path], 2)
            assert not row["stdout"] and diagnostic in row["stderr"], row
    request, report = output / "native-request.json", output / "native.json"
    save(request, dict(schema="fairyc.candidate-c-native-request/v1", output=str(output / "native"), jobs=jobs))
    run([runner, request, report])
    native = json.loads(report.read_text(encoding="utf-8"))
    assert len(native) == (2 if os.name == "nt" else 4) * len(cases)
    expected = {name: (trace, value) for name, _, _, trace, value in cases}
    for row in native:
        trace, value = expected[row["name"]]
        assert not row.get("error") and row["trap"] == 0 and row["exit"] == 0
        assert row["stdout"] == trace and row["value"] == value and row["foreignSHA256"] == digest(provider), row
    reference = []
    forwarded = []
    if sdk_process:
        for row in native:
            if row["name"] == "sdk-process-arg":
                argv = [row["image"], "한글", ""]
                q = subprocess.run(argv, cwd=PROJECT, capture_output=True, encoding="utf-8", timeout=30)
                observed = dict(argv=argv, cwd=str(PROJECT), exit=q.returncode, stdout=q.stdout,
                                stderr=q.stderr, imageSHA256=digest(Path(row["image"])))
                forwarded.append(observed)
                assert q.returncode == 0 and not q.stdout and q.stderr.strip() == "fairyc-candidate-result/v1:7", observed
    if sdk_num or sdk_process:
        # The same assertions run through the Go bootstrap's shipped SDK. Python
        # only writes fixture metadata and compares process outcomes.
        for name, first, _, trace, value in cases:
            project = output / "go-reference" / name
            (project / "src").mkdir(parents=True, exist_ok=True)
            (project / "src/main.fy").write_text(first, encoding="utf-8", newline="\n")
            (project / "fairy.toml").write_text('schema = "fairyc.manifest/v1"\n[package]\nuid = "probe.sdk"\nname = "sdk-probe"\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n', encoding="utf-8", newline="\n")
            for release in (False, True):
                argv = [str(compiler), "build"] + (["--release"] if release else [])
                q = subprocess.run(argv, cwd=project, capture_output=True, encoding="utf-8", timeout=120)
                build = dict(argv=argv, cwd=str(project), exit=q.returncode, stdout=q.stdout, stderr=q.stderr)
                reference.append(build)
                save(output / "reference-raw.json", reference)
                assert q.returncode == 0 and not q.stderr, build
                exe = project / "build" / ("sdk-probe.exe" if os.name == "nt" else "sdk-probe")
                argv = [str(exe)] + (["한글", ""] if sdk_process else [])
                q = subprocess.run(argv, cwd=project, capture_output=True, encoding="utf-8", timeout=30)
                result = dict(argv=argv, cwd=str(project), exit=q.returncode, stdout=q.stdout,
                              stderr=q.stderr, imageSHA256=digest(exe))
                reference.append(result)
                save(output / "reference-raw.json", reference)
                assert (q.returncode, q.stdout, q.stderr) == (value, trace, ""), result
    assert all(digest(Path(f["path"])) == f["sha256"] for f in fixtures)
    bootstrap = None
    if bootstrap_receipt:
        bootstrap = json.loads(bootstrap_receipt.read_text(encoding="utf-8"))
        assert bootstrap["imageSHA256"] == digest(compiler) and bootstrap["exit"] == 0
        original = Path(bootstrap["originalPath"])
        overlay = Path(bootstrap["overlayPath"])
        assert digest(original) == bootstrap["originalSHA256"]
        replacement = ("commandTimeout   = " + str(bootstrap["timeoutSeconds"]) + " * time.Second").encode("ascii")
        assert overlay.read_bytes() == original.read_bytes().replace(
            b"commandTimeout   = 60 * time.Second", replacement)
    save(receipt or ROOT / "docs/ops/evidence/candidate-module-cli-20261008.json",
         dict(schema="fairyc.candidate-module-cli/v1", passed=True, validation=rows,
              images=[dict(path=str(p), sha256=digest(p)) for p in images],
              compiler=dict(path=str(compiler), sha256=digest(compiler)),
              sdkSource=dict(path=str(sdk_source), sha256=digest(sdk_source)) if sdk_num or sdk_process else None,
              runtimeSources=[dict(path=str(p), sha256=digest(p)) for p in runtime_sources],
              validationOnlyBootstrapOverlay=bootstrap,
              runner=dict(path=str(runner), sha256=digest(runner)), fixtures=fixtures,
              native=native, forwarded=forwarded, reference=reference, sources=[dict(path=str(p.relative_to(ROOT)), sha256=digest(p))
                                      for p in sorted((PROJECT / "src").glob("*.fy"))],
              lockSHA256=digest(PROJECT / "fairy.lock"),
              limits=["explicit source-list CLI, not workspace manifest resolution",
                      "SDK aggregate instantiation pending", "Linux deferred"]))
    print(f"module CLI: {len(cases)} programs, {len(rejections)} errors, {len(native)} native C executions passed", flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--compiler", type=Path)
    parser.add_argument("--images", type=Path, nargs="+")
    parser.add_argument("--native-runner", type=Path)
    parser.add_argument("--bootstrap-receipt", type=Path)
    parser.add_argument("--receipt", type=Path)
    parser.add_argument("--phase", choices=("join", "program", "cli", "cli-generics", "cli-num", "cli-process", "repr"), required=True)
    args = parser.parse_args()
    if args.phase in ("cli", "cli-generics", "cli-num", "cli-process"):
        if not args.images or len(args.images) != 2 or not args.native_runner or not args.compiler:
            parser.error("cli phase requires two --images, --native-runner and --compiler")
        verify_cli([p.resolve() for p in args.images], args.native_runner.resolve(), args.compiler.resolve(), args.bootstrap_receipt, args.receipt, "process" if args.phase == "cli-process" else "num" if args.phase == "cli-num" else "generics" if args.phase == "cli-generics" else "modules")
        return
    if not args.compiler:
        parser.error("API phases require --compiler")
    compiler = args.compiler.resolve()
    output = ROOT / ".cache" / ("candidate-module-" + args.phase)
    output.mkdir(parents=True, exist_ok=True)
    rows = []

    def run(argv):
        result = subprocess.run(list(map(str, argv)), cwd=PROJECT,
                                capture_output=True, encoding="utf-8", timeout=480)
        row = dict(argv=list(map(str, argv)), cwd=str(PROJECT), exit=result.returncode,
                   stdout=result.stdout, stderr=result.stderr)
        rows.append(row)
        (output / "validation.json").write_text(json.dumps(rows, ensure_ascii=False,
                                                indent=2) + "\n", encoding="utf-8")
        print(result.stdout[-1600:], result.stderr[-1800:], flush=True)
        if result.returncode:
            raise RuntimeError(row)
        return result

    run([compiler, "check"])
    tests = (["tests::module_ast_join::joined_arenas_remap_ids_spans_and_tokens"]
             if args.phase == "join" else
             ["tests::repr_c_declarations::repr_c_declaration_native_contract",
              "tests::repr_c_shapes::owned_repr_c_layout_survives_hir_and_mir"]
             if args.phase == "repr" else
             ["tests::module_program::nominal_module_consumers_keep_identity_and_privacy"])
    for release in (False, True):
        argv = [compiler, "test", "--native", "--locked", "--json"]
        for test in tests:
            argv.extend(["--exact", test])
        if release:
            argv.append("--release")
        report = json.loads(run(argv).stdout)
        assert report["count"] == len(tests) and report["passed"] == len(tests) and report["status"] == "ok"

    def digest(path):
        return dict(path=str(path.relative_to(ROOT)),
                    sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    evidence = dict(schema="fairyc.candidate-module-" + args.phase + "/v1",
                    passed=True, bootstrap=digest(compiler), validation=rows,
                    sources=[digest(p) for p in sorted((PROJECT / "src").glob("*.fy"))],
                    tests=([digest(PROJECT / "tests" / "repr_c_declarations.fy"),
                            digest(PROJECT / "tests" / "repr_c_shapes.fy")]
                           if args.phase == "repr" else digest(PROJECT / "tests" /
                           ("module_ast_join.fy" if args.phase == "join" else "module_program.fy"))),
                    lock=digest(PROJECT / "fairy.lock"),
                    limits=["bounded private candidate source-set API",
                            "SDK and generic import admission pending", "Linux deferred"])
    path = ROOT / "docs/ops/evidence" / ("candidate-module-" + args.phase + "-20261008.json")
    path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
