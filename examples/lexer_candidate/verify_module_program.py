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
    ("struct", "module main;import provider as p;struct Token{i32 n;}fn i32 main(){let p::Token t=p::make();return p::value(t);}",
     "module provider;pub struct Token{i32 n;}pub fn Token make(){return Token{.n=7};}pub fn i32 value(Token t){return t.n;}", "", 7),
    ("helpers", "module main;import provider as p;fn i32 helper(){return 9;}fn i32 main(){return p::value();}",
     "module provider;fn i32 helper(){return 7;}pub fn i32 value(){return helper();}", "", 7),
    ("resource", "module main;import provider as p;@effects(console)fn i32 main(){let p::Owner x=p::make();return p::value(ref x);}",
     'module provider;@effects(console)@link_name("fixture_drop")unsafe extern C fn void dropped(i32 n);@effects(console)pub resource Owner{i32 n;cleanup{unsafe{dropped(this.n);}}}pub fn Owner make(){return Owner{.n=65};}pub fn i32 value(ref Owner x){return x.n-58;}', "A", 7),
    ("enum", "module main;import provider as p;import provider as q;fn i32 main(){let p::Kind k=q::make();return p::value(k);}",
     "module provider;pub enum Kind{Off;On;}pub fn Kind make(){return .On;}pub fn i32 value(Kind k){return switch(k){case .Off:{0}case .On:{7}};}", "", 7),
)


def verify_cli(images, runner, compiler, bootstrap_receipt):
    output = ROOT / ".cache/candidate-module-cli"
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
    provider.write_text('#include <stdint.h>\n#include <stdio.h>\nvoid fixture_drop(int32_t n){(void)fputc(n,stdout);}\n', encoding="utf-8", newline="\n")
    for name, first, second, trace, value in CLI_CASES:
        path = manifest(name, first, second)
        emitted = [run([image, "--c-modules-file", path]) for image in images]
        assert all(not row["stderr"] and row["stdout"] == emitted[0]["stdout"] for row in emitted)
        if name == "helpers":
            for image in images:
                prior = run([image, "--check-modules-file", path])
                assert prior["stdout"] == "modules ok\n" and not prior["stderr"], prior
        source = output / (name + ".observed.c")
        source.write_text('#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n' + emitted[0]["stdout"] +
                          'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t n=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",n);return 0;}\n', encoding="utf-8", newline="\n")
        jobs.append(dict(name=name, source=str(source), foreignSource=str(provider), observe=True))
    rejections = (
        ("private", "module main;import provider as p;fn i32 main(){return p::secret();}",
         "module provider;fn i32 secret(){return 7;}", "phase 3 code 6"),
        ("field", "module main;import provider as p;fn i32 main(){let p::Owner x=p::make();return x.n;}",
         "module provider;pub resource Owner{i32 n;cleanup{}}pub fn Owner make(){return Owner{.n=7};}", "phase 5 code 6201"),
        ("provider-error", "module main;import provider as p;fn i32 main(){return p::value();}",
         "module provider;pub fn i32 value(){return missing;}", "-provider.fy:1:"),
        ("no-entry", "module main;fn i32 value(){return 7;}", "module provider;", "exactly one fn i32 main"),
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
    assert len(native) == (2 if os.name == "nt" else 4) * len(CLI_CASES)
    expected = {name: (trace, value) for name, _, _, trace, value in CLI_CASES}
    for row in native:
        trace, value = expected[row["name"]]
        assert not row.get("error") and row["trap"] == 0 and row["exit"] == 0
        assert row["stdout"] == trace and row["value"] == value and row["foreignSHA256"] == digest(provider), row
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
    save(ROOT / "docs/ops/evidence/candidate-module-cli-20261008.json",
         dict(schema="fairyc.candidate-module-cli/v1", passed=True, validation=rows,
              images=[dict(path=str(p), sha256=digest(p)) for p in images],
              compiler=dict(path=str(compiler), sha256=digest(compiler)),
              validationOnlyBootstrapOverlay=bootstrap,
              runner=dict(path=str(runner), sha256=digest(runner)), fixtures=fixtures,
              native=native, sources=[dict(path=str(p.relative_to(ROOT)), sha256=digest(p))
                                      for p in sorted((PROJECT / "src").glob("*.fy"))],
              lockSHA256=digest(PROJECT / "fairy.lock"),
              limits=["explicit source-list CLI, not workspace manifest resolution",
                      "SDK/generic imports pending", "Linux deferred"]))
    print(f"module CLI: 4 programs, 4 errors, {len(native)} native C executions passed", flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--compiler", type=Path)
    parser.add_argument("--images", type=Path, nargs="+")
    parser.add_argument("--native-runner", type=Path)
    parser.add_argument("--bootstrap-receipt", type=Path)
    parser.add_argument("--phase", choices=("join", "program", "cli"), required=True)
    args = parser.parse_args()
    if args.phase == "cli":
        if not args.images or len(args.images) != 2 or not args.native_runner or not args.compiler:
            parser.error("cli phase requires two --images, --native-runner and --compiler")
        verify_cli([p.resolve() for p in args.images], args.native_runner.resolve(), args.compiler.resolve(), args.bootstrap_receipt)
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
    test = ("tests::module_ast_join::joined_arenas_remap_ids_spans_and_tokens"
            if args.phase == "join" else
            "tests::module_program::nominal_module_consumers_keep_identity_and_privacy")
    for release in (False, True):
        argv = [compiler, "test", "--native", "--locked", "--exact", test, "--json"]
        if release:
            argv.append("--release")
        report = json.loads(run(argv).stdout)
        assert report["count"] == 1 and report["passed"] == 1 and report["status"] == "ok"

    def digest(path):
        return dict(path=str(path.relative_to(ROOT)),
                    sha256=hashlib.sha256(path.read_bytes()).hexdigest())

    evidence = dict(schema="fairyc.candidate-module-" + args.phase + "/v1",
                    passed=True, bootstrap=digest(compiler), validation=rows,
                    sources=[digest(p) for p in sorted((PROJECT / "src").glob("*.fy"))],
                    tests=digest(PROJECT / "tests" / ("module_ast_join.fy" if
                                 args.phase == "join" else "module_program.fy")),
                    lock=digest(PROJECT / "fairy.lock"),
                    limits=["bounded private candidate source-set API",
                            "SDK and generic import admission pending", "Linux deferred"])
    path = ROOT / "docs/ops/evidence" / ("candidate-module-" + args.phase + "-20261008.json")
    path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
