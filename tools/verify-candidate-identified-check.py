"""Compare real identified check processes on original generic/Vec source inputs."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def snapshot(root):
    return {p.relative_to(root).as_posix(): dict(sha256=sha(p.read_bytes()), modifiedNS=p.stat().st_mtime_ns)
            for p in sorted(root.rglob('*')) if p.is_file()}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--image', type=Path, required=True)
    p.add_argument('--compiler', type=Path, required=True, help='owning Go CLI for independent lock preparation')
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--label', required=True)
    p.add_argument('--reference', type=Path)
    p.add_argument('--case', action='append', help='explicit subset; default executes every authored case')
    args = p.parse_args()
    if not __debug__ or not args.label.replace('-', '').isalnum():
        p.error('verification needs assertions and a portable label')
    image, compiler = args.image.resolve(), args.compiler.resolve()
    destination = args.output.resolve() / args.label
    destination.mkdir(parents=True, exist_ok=False)
    cases = [
        ('Scalar', 0, {'main': 'fn T id<T>(T x){return x;}fn i32 main(){return id<i32>(42);}'}, ''),
        ('Aliases', 0, {'main': 'import helper as z;import helper as a;fn i32 main(){return a::id<i32>(z::id<i32>(42));}',
                        'helper': 'pub fn T id<T>(T x){return secret<T>(x);}fn T secret<T>(T x){return x;}'}, ''),
        ('Vector', 0, {'main': 'import helper as h;import std::vec as vec;fn i32 main(){var vec::Vec<i32> v=vec::empty<i32>();assert(vec::push<i32>(ref mut v,h::id<i32>(42)).is_ok);return *vec::at<i32>(ref v,0);}',
                       'helper': 'pub fn T id<T>(T x){return x;}'}, ''),
        ('Cycle', 3, {'main': 'fn T cycle<T>(T x){return cycle<T>(x);}fn i32 main(){return cycle<i32>(42);}'}, 'E4503'),
        ('AliasCycle', 3, {'main': 'import helper as z;import helper as a;fn i32 main(){return a::cycle<i32>(z::cycle<i32>(42));}',
                           'helper': 'pub fn T cycle<T>(T x){return private_cycle<T>(x);}fn T private_cycle<T>(T x){return cycle<T>(x);}'}, 'E4503'),
        ('DuplicateParameter', 3, {'main': 'fn T id<T,T>(T x){return x;}fn i32 main(){return 0;}'}, 'E4501'),
        ('Arity', 3, {'main': 'fn T id<T>(T x){return x;}fn i32 main(){return id<i32,bool>(42);}'}, 'E4502'),
        ('Implicit', 3, {'main': 'fn T id<T>(T x){return x;}fn i32 main(){return id(42);}'}, 'E4502'),
        ('Unknown', 3, {'main': 'fn i32 main(){return missing<i32>(42);}'}, 'E0101'),
        ('Private', 3, {'main': 'import helper as h;fn i32 main(){return h::id<i32>(42);}',
                        'helper': 'fn T id<T>(T x){return x;}'}, 'E0103'),
        ('NominalArgument', 3, {'main': 'struct Leaf{i32 n;}fn T id<T>(T x){return x;}fn i32 main(){let Leaf v=id<Leaf>(Leaf{.n=42});return v.n;}'}, 'E4502'),
        ('NonGenericApplication', 3, {'main': 'fn i32 id(i32 x){return x;}fn i32 main(){return id<i32>(42);}'}, ''),
        ('NonGenericImported', 3, {'main': 'import helper as h;fn i32 main(){return h::id<i32>(42);}', 'helper': 'pub fn i32 id(i32 x){return x;}'}, ''),
        ('NonGenericWithTemplate', 3, {'main': 'fn T unused<T>(T x){return x;}fn i32 id(i32 x){return x;}fn i32 main(){return id<i32>(42);}'}, 'E0101'),
        ('UnknownWithTemplate', 3, {'main': 'fn T unused<T>(T x){return x;}fn i32 main(){return missing<i32>(42);}'}, 'E0101'),
        ('ReferenceArgument', 3, {'main': 'fn T id<T>(T x){return x;}fn i32 main(){return id<ref i32>(42);}'}, 'E4502'),
        ('UnsupportedScalar', 3, {'main': 'fn T id<T>(T x){return x;}fn i32 main(){return id<u16>(42);}'}, ''),
    ]
    total_cases = len(cases)
    if args.case:
        assert len(set(args.case)) == len(args.case) and set(args.case) <= {c[0] for c in cases}
        cases = [case for case in cases if case[0] in args.case]
    expected = {} if args.reference is None else {r['name']: r for r in json.loads(args.reference.read_text())['cases']}
    observations = []
    for name, status, modules, code in cases:
        root = destination / name
        (root / 'src').mkdir(parents=True)
        (root / 'fairy.toml').write_text("schema='fairyc.manifest/v1'\n[package]\nuid='example.identified.check'\nname='identified-check'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n", encoding='utf-8', newline='\n')
        for module, body in modules.items():
            # Korean trivia and CRLF exercise original byte/line provenance.
            (root / 'src' / (module + '.fy')).write_bytes(('module ' + module + '; // 한글\r\n' + body).encode())
        prep = subprocess.run([str(compiler), 'check'], cwd=root, capture_output=True, timeout=120)
        assert prep.returncode == status and (root / 'fairy.lock').is_file(), (name, prep)
        before = snapshot(root)
        started = time.monotonic()
        result = subprocess.run([str(image), 'check', '--locked'], cwd=root, capture_output=True, timeout=240)
        assert snapshot(root) == before, (name, 'locked check changed source/lock bytes or modification times')
        diagnostic = re.search(rb'(?m)^(.+?):([0-9]+):([0-9]+): (E[0-9]{4}):.*?\[bytes ([0-9]+)\.\.([0-9]+)', result.stderr)
        first = None if diagnostic is None else [s.decode('utf-8') for s in diagnostic.groups()]
        unsupported = result.stderr.decode('utf-8').splitlines() if result.stderr.startswith(b'fairyc: semantic subset does not support ') else []
        row = dict(name=name, exit=result.returncode, stdout=result.stdout.decode('utf-8'), stderr=result.stderr.decode('utf-8'),
                   firstDiagnostic=first, unsupportedDiagnostics=unsupported, seconds=time.monotonic()-started,
                   inputs={path: value['sha256'] for path, value in before.items()},
                   expectedExit=status, expectedCode=code)
        row['matched'] = row['exit'] == status and (not code or first is not None and first[3] == code)
        if expected:
            oracle = expected[name]
            row['matched'] = row['matched'] and row['inputs'] == oracle['inputs'] and row['stdout'] == oracle['stdout'] and row['firstDiagnostic'] == oracle['firstDiagnostic'] and row['unsupportedDiagnostics'] == oracle['unsupportedDiagnostics']
        observations.append(row)
        report = dict(schema='fairyc.candidate-identified-check/v1', image=dict(path=str(image), sha256=sha(image.read_bytes())),
                      lockCompiler=dict(path=str(compiler), sha256=sha(compiler.read_bytes())),
                      comparison='exact stdout, status and first diagnostic path/line/column/code/byte span or complete unsupported diagnostics; locked bytes/mtime unchanged',
                      selectedCases=[case[0] for case in cases], allCases=total_cases,
                      cases=observations, passed=all(r['matched'] for r in observations))
        (destination / 'observations.json').write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8', newline='\n')
        print(name, result.returncode, 'matched=' + str(row['matched']), flush=True)
    assert all(row['matched'] for row in observations), json.dumps([r for r in observations if not r['matched']], indent=2)


if __name__ == '__main__':
    main()
