#!/usr/bin/env python3
"""Result fixtures and observations; all compiler behavior runs in Fairy."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parent
SCALARS = ('i32', 'i64', 'u8', 'u32', 'usize', 'bool')
TYPE_POSITIVES = [
    (f'pair-{success}-{failure}',
     f'fn Result<{success},{failure}> id(Result<{success},{failure}> x){{let Result<{success},{failure}> y=x;return y;}}')
    for success in SCALARS for failure in SCALARS
]
TYPE_POSITIVES += [
    ('calls', 'fn Result<i64,bool> id(Result<i64,bool> x){return x;}fn Result<i64,bool> call(Result<i64,bool> x){return id(x);}'),
    ('if', 'fn Result<i32,bool> choose(bool b,Result<i32,bool> x,Result<i32,bool> y){return if(b){x}else{y};}'),
    ('assignment', 'fn Result<i32,bool> f(Result<i32,bool> x){var Result<i32,bool> y=x;y=x;return y;}'),
    ('coexist', 'enum E{A;}struct S{i32 n;}fn Option<i32> o(Option<i32> x){return x;}fn Result<i32,i32> r(Result<i32,i32> x){return x;}'),
]
TYPE_NEGATIVES = [
    ('wrong-success', 'fn Result<i64,bool> f(Result<i32,bool> x){return x;}', 'type mismatch'),
    ('wrong-error', 'fn Result<i32,i64> f(Result<i32,bool> x){return x;}', 'type mismatch'),
    ('reversed', 'fn Result<bool,i32> f(Result<i32,bool> x){return x;}', 'type mismatch'),
    ('wrong-argument', 'fn Result<i32,bool> id(Result<i32,bool> x){return x;}fn Result<i32,bool> f(Result<i64,bool> x){return id(x);}', 'type mismatch'),
    ('wrong-assignment', 'fn Result<i32,bool> f(Result<i32,bool> x,Result<i64,bool> z){var Result<i32,bool> y=x;y=z;return y;}', 'type mismatch'),
    ('option', 'fn Result<i32,i32> f(Option<i32> x){return x;}', 'type mismatch'),
    ('scalar', 'fn i32 f(Result<i32,bool> x){return x;}', 'type mismatch'),
    ('arithmetic', 'fn Result<i32,bool> f(Result<i32,bool> x){return x+x;}', 'invalid scalar operator'),
    ('comparison', 'fn bool f(Result<i32,bool> x){return x==x;}', 'invalid scalar operator'),
    ('field-storage', 'struct Box{Result<i32,bool> value;}fn i32 main(){return 0;}', 'invalid nominal type'),
    ('one-argument', 'fn Result<i32> f(){return 0;}', None),
    ('three-arguments', 'fn Result<i32,bool,u8> f(){return 0;}', None),
    ('void-success', 'fn Result<void,bool> f(){return 0;}', None),
    ('void-error', 'fn Result<i32,void> f(){return 0;}', None),
    ('unknown', 'fn Result<Missing,bool> f(){return 0;}', None),
    ('qualified', 'fn unknown::Result<i32,bool> f(){return 0;}', None),
    ('nested', 'fn Result<Option<i32>,bool> f(){return 0;}', None),
]
# Every admitted pair must differ when either component changes. This checks
# actual candidate source typing, not only the private numeric encoding helper.
for index, success in enumerate(SCALARS):
    for error_index, failure in enumerate(SCALARS):
        changed_success = SCALARS[(index + 1) % len(SCALARS)]
        changed_error = SCALARS[(error_index + 1) % len(SCALARS)]
        TYPE_NEGATIVES.append((f'changed-success-{success}-{failure}',
                               f'fn Result<{success},{failure}> f(Result<{changed_success},{failure}> x){{return x;}}',
                               'type mismatch'))
        TYPE_NEGATIVES.append((f'changed-error-{success}-{failure}',
                               f'fn Result<{success},{failure}> f(Result<{success},{changed_error}> x){{return x;}}',
                               'type mismatch'))

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--images', nargs='+', required=True)
    parser.add_argument('--phase', choices=['types'], required=True)
    parser.add_argument('--receipt', required=True)
    args = parser.parse_args()
    images = [Path(path).resolve() for path in args.images]
    receipt = Path(args.receipt).resolve()
    receipt.parent.mkdir(parents=True, exist_ok=True)
    rows, fixtures = [], []
    passed = False
    try:
        for rejected, cases in ((False, TYPE_POSITIVES), (True, TYPE_NEGATIVES)):
            for case in cases:
                name, source = case[:2]
                expected = 2 if rejected else 0
                path = receipt.parent / (name + '.fy')
                data = ('module main;' + source).encode('utf-8')
                path.write_bytes(data)
                digest = sha(path)
                fixtures.append(dict(name=name, inputHex=data.hex(), sha256=digest, exit=expected))
                outputs = []
                for image in images:
                    argv = [str(image), '--check-types-file', str(path)]
                    p = subprocess.run(argv, cwd=ROOT, capture_output=True, encoding='utf-8', timeout=30)
                    row = dict(argv=argv, cwd=str(ROOT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr)
                    rows.append(row)
                    assert p.returncode == expected, row
                    if rejected:
                        assert not p.stdout and p.stderr, row
                        if case[2]:
                            assert case[2] in p.stderr, row
                    else:
                        assert p.stdout == 'types ok\n' and not p.stderr, row
                    assert sha(path) == digest
                    outputs.append((p.returncode, p.stdout, p.stderr))
                assert len(set(outputs)) == 1
        passed = True
    finally:
        report = dict(schema='fairyc.result-candidate-verification/v1', phase=args.phase, passed=passed,
                      images=[dict(path=str(path), sha256=sha(path)) for path in images],
                      cli=rows, fixtures=fixtures,
                      sources=[dict(path=str(path.relative_to(ROOT)), sha256=sha(path))
                               for path in sorted((ROOT / 'src').glob('*.fy'))])
        receipt.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8', newline='\n')
    print(f'PASS: Result {args.phase}; {len(rows)} candidate invocations')

if __name__ == '__main__':
    main()
