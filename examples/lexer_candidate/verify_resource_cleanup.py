#!/usr/bin/env python3
"""Observe cleanup type/ownership checks; scripts do not implement cleanup."""
import argparse, hashlib, json, subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent
CASES = [
    ('read-write', 'resource R{i32 n;cleanup{let i32 x=this.n;this.n=x+1;}}fn i32 main(){return 0;}', None),
    ('nested-shadow', 'struct P{i32 n;}resource R{P p;cleanup{{let i32 this=4;let i32 x=this;};let i32 y=this.p.n;}}', None),
    ('nested-write', 'struct P{i32 n;}resource R{P p;cleanup{this.p.n=3;}}', 'assignment target is immutable'),
    ('loop-return', 'resource R{i32 n;cleanup{var i32 i=0;while(i<2){this.n+=1;i+=1;}return;}}', None),
    ('local-transfer', 'resource S{cleanup{}}resource R{cleanup{let S s=S{};take(s);}}fn void take(S s){}', None),
    ('move-call', 'resource R{cleanup{take(this);}}fn void take(R r){}', 'E1102'),
    ('move-binding', 'resource R{cleanup{let R other=this;}}', 'E1102'),
    ('overwrite-receiver', 'resource R{cleanup{this=R{};}}', 'E1102'),
    ('partial-move', 'resource S{cleanup{}}resource R{S s;cleanup{take(this.s);}}fn void take(S s){}', 'E1103'),
    ('wrong-field-type', 'resource R{i32 n;cleanup{this.n=true;}}', 'type mismatch'),
    ('return-value', 'resource R{cleanup{return 1;}}', 'type mismatch'),
    ('tail-value', 'resource R{cleanup{1}}', 'expected ;'),
    ('fallible-cleanup', 'resource R{cleanup{let i32 x=f()?;}}fn Result<i32,bool> f(){return .ok(1);}', '? requires enclosing Result'),
    ('local-reuse', 'resource S{cleanup{}}resource R{cleanup{let S s=S{};take(s);take(s);}}fn void take(S s){}', 'E1101'),
    ('local-result', 'resource R{cleanup{let Result<i32,bool> x=f();}}fn Result<i32,bool> f(){return .ok(1);}', 'Result value must be handled'),
    ('out-of-scope', 'resource R{i32 n;cleanup{let i32 x=this.n;}}fn void f(){let i32 n=this.n;}', 'unresolved'),
    ('unknown-field', 'resource R{cleanup{let i32 n=this.missing;}}', 'unknown field'),
]

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--images', nargs='+', required=True)
    ap.add_argument('--receipt', required=True)
    args = ap.parse_args()
    images = [Path(p).resolve() for p in args.images]
    dest = Path(args.receipt).resolve()
    dest.parent.mkdir(parents=True, exist_ok=True)
    rows, fixtures, passed = [], [], False
    try:
        for name, source, diagnostic in CASES:
            path = dest.parent / (name + '.fy')
            path.write_text('module main;' + source, encoding='utf-8', newline='\n')
            fixtures.append(dict(name=name, inputHex=path.read_bytes().hex(), sha256=sha(path), diagnostic=diagnostic))
            for image in images:
                modes = ['--check-types-file'] if diagnostic else ['--check-types-file', '--hir-file', '--mir-file', '--c-file']
                for mode in modes:
                    argv = list(map(str, [image, mode, path]))
                    p = subprocess.run(argv, cwd=ROOT, capture_output=True, encoding='utf-8', timeout=120)
                    row = dict(argv=argv, cwd=str(ROOT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr)
                    rows.append(row)
                    expected = 0 if not diagnostic and mode == '--check-types-file' else 2
                    assert p.returncode == expected, row
                    if expected == 0:
                        assert p.stdout == 'types ok\n' and not p.stderr, row
                    else:
                        assert not p.stdout and p.stderr, row
                        if diagnostic:
                            assert diagnostic in p.stderr, row
                assert sha(path) == fixtures[-1]['sha256']
        passed = True
    finally:
        dest.write_text(json.dumps(dict(schema='fairyc.resource-cleanup-check/v1', unit=21, passed=passed,
            images=[dict(path=str(p), sha256=sha(p)) for p in images], fixtures=fixtures, cli=rows,
            sources=[dict(path=str(p.relative_to(ROOT)), sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],
            executionBoundary='Cleanup source type/ownership checks are active. Nonempty cleanup HIR/MIR/C fails closed until unit 22 owns private glue.'), ensure_ascii=False, indent=2)+'\n', encoding='utf-8', newline='\n')
    print(f'PASS cleanup: {len(fixtures)} fixtures; {len(rows)} CLI observations')

if __name__ == '__main__':
    main()
