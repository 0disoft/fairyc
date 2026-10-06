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
    ('if', 'fn Result<i32,bool> choose(bool b,Result<i32,bool> x,Result<i32,bool> y){return if(b){discard y;x}else{discard x;y};}'),
    ('assignment', 'fn Result<i32,bool> f(Result<i32,bool> x){var Result<i32,bool> y=x;discard y;y=x;return y;}'),
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
    ('nested-void', 'fn Result<Option<void>,bool> f(){return 0;}', None),
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

VALUES = dict(zip(SCALARS, ('-2147483648', '-9223372036854775808', '255', '4294967295', '18446744073709551615', 'true')))
PATTERN_POSITIVES = [
    (f'construct-{success}-{failure}',
     f'fn Result<{success},{failure}> make(bool yes){{return if(yes){{.ok({VALUES[success]})}}else{{.err({VALUES[failure]})}};}}'
     f'fn i32 main(){{return switch(make(true)){{case .ok(n):{{assert(n=={VALUES[success]});0}}case .err(e):{{assert(e=={VALUES[failure]});1}}}};}}')
    for success in SCALARS for failure in SCALARS
]
PATTERN_POSITIVES += [
    ('error-first', 'fn Result<i64,bool> make(){return .err(true);}fn i32 main(){return switch(make()){case .err(e):{if(e){7}else{0}}case .ok(n):{assert(n==0);1}};}'),
    ('discard-captures', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(_):{1}case .err(_):{2}};}'),
    ('shadow-captures', 'fn i32 f(Result<i32,bool> r){let i32 n=4;return switch(r){case .ok(n):{n}case .err(n):{if(n){1}else{0}}}+n;}'),
    ('return-arm-first', 'fn i32 f(Result<i64,bool> r){return switch(r){case .err(e):{return if(e){2}else{3};}case .ok(n):{assert(n==4);7}};}'),
    ('return-arm-last', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(_):{return 2;}};}'),
    ('both-return', 'fn i32 f(Result<i32,bool> r){switch(r){case .ok(n):{return n;}case .err(_):{return 2;}};return 3;}'),
    ('assert-arm-first', 'fn i32 f(Result<i32,bool> r){let n=switch(r){case .err(_):{assert(false);}case .ok(n):{n}};return n;}'),
    ('assert-arm-last', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(_):{assert(false);}};}'),
    ('context-argument', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(e):{if(e){1}else{2}}};}fn i32 main(){return f(.err(true));}'),
    ('context-assignment', 'fn i32 main(){var Result<i32,bool> r=.ok(3);discard r;r=.err(true);return switch(r){case .ok(n):{n}case .err(e):{if(e){1}else{2}}};}'),
    ('option-coexist', 'fn i32 main(){let Option<i32> o=.some(3);let Result<i32,bool> r=.ok(2);return switch(r){case .ok(n):{n}case .err(_):{0}}+switch(o){case .some(n):{n}case .none:{0}};}'),
]
PATTERN_NEGATIVES = [
    ('ok-no-context', 'fn i32 main(){let r=.ok(1);return 0;}', 'Result context required'),
    ('err-no-context', 'fn i32 main(){let r=.err(true);return 0;}', 'Result context required'),
    ('ok-bare', 'fn Result<i32,bool> f(){return .ok;}', 'invalid Result construction'),
    ('err-bare', 'fn Result<i32,bool> f(){return .err;}', 'invalid Result construction'),
    ('ok-empty', 'fn Result<i32,bool> f(){return .ok();}', 'invalid Result construction'),
    ('err-two', 'fn Result<i32,bool> f(){return .err(true,false);}', 'invalid Result construction'),
    ('wrong-ok', 'fn Result<i32,bool> f(){return .ok(true);}', 'type mismatch'),
    ('wrong-err', 'fn Result<i32,bool> f(){return .err(1);}', 'type mismatch'),
    ('ok-overflow', 'fn Result<u8,bool> f(){return .ok(256);}', 'integer literal out of range'),
    ('err-overflow', 'fn Result<bool,u8> f(){return .err(256);}', 'integer literal out of range'),
    ('unknown-construction', 'fn Result<i32,bool> f(){return .other(1);}', 'invalid Result construction'),
    ('some-is-not-ok', 'fn Result<i32,bool> f(){return .some(1);}', 'invalid Result construction'),
    ('none-is-not-err', 'fn Result<i32,bool> f(){return .none;}', 'invalid Result construction'),
    ('missing-ok', 'fn i32 f(Result<i32,bool> r){return switch(r){case .err(_):{0}};}', 'non-exhaustive Result switch'),
    ('missing-err', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}};}', 'non-exhaustive Result switch'),
    ('duplicate-ok', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .ok(_):{1}case .err(_):{0}};}', 'duplicate switch case'),
    ('ok-no-binding', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok:{0}case .err(_):{1}};}', 'invalid Result pattern'),
    ('err-no-binding', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err:{1}};}', 'invalid Result pattern'),
    ('unknown-pattern', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .bad(_):{1}};}', 'invalid Result pattern'),
    ('immutable-ok', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n=2;n}case .err(_):{0}};}', 'assignment target is immutable'),
    ('immutable-err', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(e):{e=false;0}};}', 'assignment target is immutable'),
    ('arm-leak', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(_):{n}};}', 'unresolved name'),
    ('capture-escapes', 'fn i32 f(Result<i32,bool> r){discard switch(r){case .ok(n):{n}case .err(_):{0}};return n;}', 'unresolved name'),
    ('join-mismatch', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(e):{e}};}', 'type mismatch'),
    ('wrong-return-arm', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(e):{return e;}};}', 'type mismatch'),
    ('assert-true-is-value-less', 'fn i32 f(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(_):{assert(true);}};}', 'type mismatch'),
]

PROPAGATION_POSITIVES = [
    (f'propagate-{success}-{failure}',
     f'fn Result<bool,{failure}> f(Result<{success},{failure}> r){{let {success} n=r?;return .ok(true);}}')
    for success in SCALARS for failure in SCALARS
]
PROPAGATION_POSITIVES += [
    ('propagate-call', 'fn Result<i64,bool> value(){return .ok(4000000000);}fn Result<i64,bool> f(){return .ok(value()?);}'),
    ('propagate-group', 'fn Result<i32,bool> f(Result<i32,bool> r){return .ok((r)?);}'),
    ('propagate-argument', 'fn i32 id(i32 n){return n;}fn Result<i32,bool> f(Result<i32,bool> r){return .ok(id(r?));}'),
    ('propagate-branch', 'fn Result<i32,bool> f(Result<i32,bool> r,bool yes){return .ok(if(yes){r?}else{discard r;0});}'),
    ('propagate-loop', 'fn Result<i32,bool> f(Result<i32,bool> r){var i32 n=0;while(n<1){n=r?;}discard r;return .ok(n);}'),
    ('propagate-defer-capture', 'fn void log(i32 n){}fn Result<i32,bool> f(Result<i32,bool> r){defer log(r?);return .ok(0);}'),
    ('propagate-condition', 'fn Result<i32,bool> f(Result<bool,bool> r){if(r?){return .ok(1);}return .ok(0);}'),
]
PROPAGATION_NEGATIVES = [
    ('propagate-scalar', 'fn Result<i32,bool> f(i32 n){return .ok(n?);}', '? requires Result operand'),
    ('propagate-option', 'fn Result<i32,bool> f(Option<i32> r){return .ok(r?);}', '? requires Result operand'),
    ('propagate-enum', 'enum E{A;}fn Result<i32,bool> f(E r){discard r?;return .ok(0);}', '? requires Result operand'),
    ('propagate-scalar-return', 'fn i32 f(Result<i32,bool> r){return r?;}', '? requires enclosing Result with identical error type'),
    ('propagate-void-return', 'fn void f(Result<i32,bool> r){discard r?;}', '? requires enclosing Result with identical error type'),
    ('propagate-option-return', 'fn Option<i32> f(Result<i32,bool> r){return .some(r?);}', '? requires enclosing Result with identical error type'),
    ('propagate-reversed', 'fn Result<bool,i32> f(Result<i32,bool> r){discard r?;return .ok(true);}', '? requires enclosing Result with identical error type'),
    ('propagate-no-coercion', 'fn Result<i32,i64> f(Result<i32,i32> r){return .ok(r?);}', '? requires enclosing Result with identical error type'),
    ('propagate-return-value', 'fn Result<i32,bool> f(Result<i32,bool> r){return r?;}', 'type mismatch'),
    ('propagate-wrong-binding', 'fn Result<i32,bool> f(Result<i64,bool> r){let i32 n=r?;return .ok(n);}', 'type mismatch'),
    ('propagate-twice', 'fn Result<i32,bool> f(Result<i32,bool> r){return .ok(r??);}', '? requires Result operand'),
    ('propagate-no-inference', 'fn Result<i32,bool> f(){return .ok(.ok(1)?);}', 'Result context required'),
    ('propagate-for-init', 'fn Result<i32,bool> f(Result<i32,bool> r){for(let i32 n=r?;n<1;n+=1){}return .ok(0);}', 'unsupported for header'),
    ('propagate-for-condition', 'fn Result<i32,bool> f(Result<bool,bool> r){for(let i32 n=0;r?;n+=1){}return .ok(0);}', 'unsupported for header'),
    ('propagate-for-post', 'fn Result<i32,bool> f(Result<i32,bool> r){for(var i32 n=0;n<1;n=r?){}return .ok(0);}', 'unsupported for header'),
    ('propagate-unresolved', 'fn Result<i32,bool> f(){return .ok(missing?);}', 'unresolved name'),
]
for success in SCALARS:
    for index, failure in enumerate(SCALARS):
        other = SCALARS[(index+1) % len(SCALARS)]
        PROPAGATION_NEGATIVES.append((f'propagate-error-{success}-{failure}',
            f'fn Result<{success},{other}> f(Result<{success},{failure}> r){{discard r?;return .ok({VALUES[success]});}}',
            '? requires enclosing Result with identical error type'))

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--images', nargs='+', required=True)
    parser.add_argument('--phase', choices=['types', 'patterns', 'hir', 'propagation-types'], required=True)
    parser.add_argument('--receipt', required=True)
    args = parser.parse_args()
    images = [Path(path).resolve() for path in args.images]
    receipt = Path(args.receipt).resolve()
    receipt.parent.mkdir(parents=True, exist_ok=True)
    rows, fixtures = [], []
    passed = False
    try:
        positives = TYPE_POSITIVES + (PATTERN_POSITIVES if args.phase != 'types' else [])
        negatives = TYPE_NEGATIVES + (PATTERN_NEGATIVES if args.phase != 'types' else [])
        if args.phase == 'propagation-types':
            positives, negatives = PROPAGATION_POSITIVES, PROPAGATION_NEGATIVES
        modes = ['--check-types-file'] + (['--hir-file'] if args.phase == 'hir' else [])
        for rejected, cases in ((False, positives), (True, negatives)):
            for case in cases:
                name, source = case[:2]
                expected = 2 if rejected else 0
                path = receipt.parent / (name + '.fy')
                data = ('module main;' + source).encode('utf-8')
                path.write_bytes(data)
                digest = sha(path)
                fixtures.append(dict(name=name, inputHex=data.hex(), sha256=digest, exit=expected))
                for mode in modes:
                    outputs = []
                    for image in images:
                        argv = [str(image), mode, str(path)]
                        p = subprocess.run(argv, cwd=ROOT, capture_output=True, encoding='utf-8', timeout=30)
                        row = dict(argv=argv, cwd=str(ROOT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr)
                        rows.append(row)
                        assert p.returncode == expected, row
                        if rejected:
                            assert not p.stdout and p.stderr, row
                            if case[2]:
                                assert case[2] in p.stderr, row
                        elif mode == '--hir-file':
                            assert p.stdout.startswith('fairyc.enum-payload-hir/v1\n') and 'result:' in p.stdout and not p.stderr, row
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
