"""Compare native trap locations with Go, including package and snapshot identity."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', required=True, type=Path)
    parser.add_argument('--reference', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--profile', choices=['development', 'release'])
    parser.add_argument('--case', action='append', help='run only these named cases')
    args = parser.parse_args()
    assert __debug__
    build = json.loads(args.build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    assert all(sha(ROOT / s['path']) == s['sha256'] for s in build['sources'])
    receipt_sha, reference_sha = sha(args.build_receipt), sha(args.reference)
    verifier_sha = sha(Path(__file__))
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    manifest = ('schema = "fairyc.manifest/v1"\n[package]\nuid = "example.trap-location"\n'
                'name = "trap-location"\nversion = "1.0.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n')
    cases = {
        'assertion': 'module main;\nfn i32 main(){\n    assert(false);return 0;\n}\n',
        'unicode': 'module main;\nfn i32 main(){\n    let label="한글🧚"; assert(label.len==10); assert(false);return 0;\n}\n',
        'bounds': 'module main;fn usize index(){return 2;}\nfn i32 main(){let i32[2] data=[1,2];return data[index()];}\n',
        'slice': 'module main;fn usize index(){return 3;}\nfn i32 main(){let u8[2] data=[1,2];let ref u8[] view=ref data;discard view[..index()];return 0;}\n',
        'compound-overflow': 'module main;\nfn i32 main(){var u8 n=255;n+=1;return 0;}\n',
        'negation-overflow': 'module main;\nfn i32 main(){var i32 n=-2147483648;return -n;}\n',
        'division-zero': 'module main;\nfn i32 main(){var i32 n=0;return 1/n;}\n',
        'division-overflow': 'module main;\nfn i32 main(){var i32 n=-2147483648;return n / -1;}\n',
        'shift': 'module main;\nfn i32 main(){var u32 n=32;discard n<<n;return 0;}\n',
        'dependency': 'module main;import dep as dep;fn i32 main(){return dep::value();}\n',
        'long-source-key': None,
        'source-replaced': None,
    }
    assert not args.case or set(args.case) <= cases.keys(), 'unknown case'
    rows = []
    for image in build['images']:
        if args.profile and image['profile'] != args.profile:
            continue
        candidate = Path(image['path'])
        assert sha(candidate) == image['sha256']
        assert sha(candidate.with_name('fairyc-candidate-build.exe')) == image['driverSHA256']
        for release in (False, True):
            flags = ['--release'] if release else []
            for name, source in cases.items():
                if args.case and name not in args.case:
                    continue
                observed = []
                for engine, binary in [('go', args.reference.resolve()), ('candidate', candidate)]:
                    home = output / image['profile'] / ('release' if release else 'development') / name / engine
                    (home / 'src').mkdir(parents=True)
                    (home / 'fairy.toml').write_text(manifest, encoding='utf-8')
                    if name == 'dependency':
                        with (home / 'fairy.toml').open('a', encoding='utf-8') as f:
                            f.write('\n[workspace]\nmembers = ["dep"]\n[dependencies.dep]\npath = "dep"\nversion = "1.0.0"\nfeatures = []\n')
                        (home / 'dep/src').mkdir(parents=True)
                        (home / 'dep/fairy.toml').write_text(manifest.replace('example.trap-location', 'example.trap-provider'), encoding='utf-8')
                        (home / 'dep/src/main.fy').write_text('module main;pub fn i32 value(){\n    assert(false);return 0;\n}\n', encoding='utf-8')
                        (home / 'dep/src/dep.fy').write_text('module dep;import main as h;pub fn i32 value(){return h::value();}\n', encoding='utf-8')
                    if name == 'source-replaced':
                        path = json.dumps(str(home / 'src/main.fy'), ensure_ascii=False)
                        source = ('module main;import std::fs as fs;\n'
                                  '@effects(filesystem) fn i32 main(){\n'
                                  'let file=switch(fs::create_truncate(c' + path + ')){case .ok(p):{p}case .err(_):{return 91;}};\n'
                                  'assert(fs::close(file).is_ok);\n'
                                  'assert(false);return 0;\n}\n')
                    if name == 'long-source-key':
                        module = 'a' * 251
                        source = f'module main;import {module} as h;fn i32 main(){{return h::value();}}\n'
                        (home / 'src' / (module + '.fy')).write_text(
                            f'module {module};pub fn i32 value(){{assert(false);return 0;}}\n', encoding='utf-8')
                    (home / 'src/main.fy').write_text(source, encoding='utf-8', newline='\n')
                    result = subprocess.run([str(binary), 'run', *flags], cwd=home,
                                            capture_output=True, timeout=180)
                    row = dict(engine=engine, exit=result.returncode, stdout=result.stdout.decode('utf-8'), stderr=result.stderr.decode('utf-8'))
                    observed.append(row)
                    if name == 'source-replaced' and result.returncode == 5:
                        assert (home / 'src/main.fy').read_bytes() == b'', 'fixture did not replace its source'
                matched = all(observed[0][k] == observed[1][k] for k in ('exit', 'stdout', 'stderr'))
                valid = (observed[0]['exit'] == 5 and not observed[0]['stdout'] and
                         observed[0]['stderr'].startswith('trap: ') and
                         ('\n  at ' in observed[0]['stderr']) == (name != 'long-source-key'))
                rows.append(dict(compilerProfile=image['profile'], programProfile='release' if release else 'development', case=name,
                                 exactMatch=matched, validReference=valid, observations=observed))
                (output / 'observations.json').write_text(json.dumps(rows, indent=2) + '\n', encoding='utf-8')
                print('PASS' if matched and valid else 'FAIL', image['profile'], flags, name, flush=True)
    assert rows
    assert sha(args.build_receipt) == receipt_sha and sha(args.reference) == reference_sha
    assert sha(Path(__file__)) == verifier_sha
    assert all(sha(ROOT / s['path']) == s['sha256'] for s in build['sources'])
    passed = all(r['exactMatch'] and r['validReference'] for r in rows)
    (output / 'receipt.json').write_text(json.dumps(dict(passed=passed, comparisons=len(rows),
        buildReceiptSHA256=receipt_sha, referenceSHA256=reference_sha, verifierSHA256=verifier_sha, rows=rows), indent=2) + '\n', encoding='utf-8')
    if not passed:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
