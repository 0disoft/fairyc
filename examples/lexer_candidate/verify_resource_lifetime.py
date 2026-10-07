#!/usr/bin/env python3
"""Bounded fixtures and native observations; all lifetime logic is Fairy code."""
import argparse, hashlib, json, re, subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parents[1]
PREFIX = 'module main;fn void mark(i32 n){}'
CASES = {
    22: [
        ('lifo', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R a=R{.n=65};let R b=R{.n=66};};return 0;}', 'BA', 0, 0),
        ('nested', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R a=R{.n=65};{let R b=R{.n=66};let R c=R{.n=67};};};return 0;}', 'CBA', 0, 0),
        ('move-binding', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R a=R{.n=65};let R b=a;};return 0;}', 'A', 0, 0),
        ('call-transfer', 'resource R{i32 n;cleanup{mark(this.n);}}fn void take(R r){mark(88);}fn i32 main(){{let R a=R{.n=65};take(a);};return 0;}', 'XA', 0, 0),
        ('receiver-store', 'resource R{i32 n;cleanup{this.n+=1;mark(this.n);}}fn i32 main(){{let R a=R{.n=65};};return 0;}', 'B', 0, 0),
        ('empty', 'resource R{cleanup{}}fn void take(R r){}fn i32 main(){{let R a=R{};let R b=a;take(b);};return 0;}', '', 0, 0),
        ('shadow', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R r=R{.n=65};{let R r=R{.n=66};};};return 0;}', 'BA', 0, 0),
        ('wide-field', 'resource R{i64 n;cleanup{assert(this.n==4000000000);mark(65);}}fn i32 main(){{let R r=R{.n=4000000000};assert(r.n==4000000000);};return 7;}', 'A', 7, 0),
    ],
    23: [
        ('return-root', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R r=R{.n=65};return 7;}', 'A', 7, 0),
        ('return-nested', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R a=R{.n=65};{let R b=R{.n=66};return 9;};return 0;}', 'BA', 9, 0),
        ('return-frozen', 'resource R{i32 n;cleanup{this.n+=1;mark(this.n);}}fn i32 main(){let R r=R{.n=65};return r.n;}', 'B', 65, 0),
        ('return-donor', 'resource R{i32 n;cleanup{mark(this.n);}}fn R make(){return R{.n=65};}fn R id(R r){return r;}fn i32 main(){let R r=id(make());return 7;}', 'A', 7, 0),
        ('return-if', 'resource R{i32 n;cleanup{mark(this.n);}}fn R select(bool yes){return if(yes){R{.n=65}}else{R{.n=66}};}fn i32 main(){let R a=select(true);let R b=select(false);return 0;}', 'BA', 0, 0),
        ('return-switch', 'resource R{i32 n;cleanup{mark(this.n);}}enum E{A;B;}fn R select(E e){return switch(e){case .A:{R{.n=65}}case .B:{R{.n=66}}};}fn i32 main(){let R r=select(.B);return 0;}', 'B', 0, 0),
        ('question-ok', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> f(bool ok){let R r=R{.n=65};let Result<i32,bool> input=if(ok){.ok(7)}else{.err(true)};let i32 n=input?;mark(88);return .ok(n);}fn i32 main(){return switch(f(true)){case .ok(n):{n}case .err(_):{99}};}', 'XA', 7, 0),
        ('question-err', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> f(bool ok){let R r=R{.n=65};let Result<i32,bool> input=if(ok){.ok(7)}else{.err(true)};let i32 n=input?;mark(88);return .ok(n);}fn i32 main(){return switch(f(false)){case .ok(n):{n}case .err(e):{if(e){1}else{2}}};}', 'A', 1, 0),
        ('short-return', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 f(bool yes){let R r=R{.n=65};discard yes && {return 7;true};return 8;}fn i32 main(){return f(false)+f(true);}', 'AA', 15, 0),
        ('cleanup-local', 'resource S{cleanup{mark(83);}}resource R{cleanup{let S s=S{};mark(82);return;}}fn i32 main(){let R r=R{};return 0;}', 'RS', 0, 0),
    ],
    27: [
        ('wildcard-scope-order', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.some(R{.n=65});discard switch(x){case .some(_):{let R b=R{.n=66};mark(88);}case .none:{mark(89);}};return 0;}', 'XBA', 0, 0),
        ('wildcard-continue', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){var i32 n=0;while(n<2){let Option<R> x=.some(R{.n=65});n+=1;discard switch(x){case .some(_):{let R b=R{.n=66};continue;}case .none:{break;}};}return 0;}', 'BABA', 0, 0),
        ('option-some-move', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.some(R{.n=65});let Option<R> y=x;return switch(y){case .some(r):{r.n}case .none:{0}};}', 'A', 65, 0),
        ('option-none', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.none;return 0;}', '', 0, 0),
        ('option-wildcard', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.some(R{.n=65});discard switch(x){case .some(_):{mark(88);}case .none:{mark(89);}};return 0;}', 'XA', 0, 0),
        ('option-nested', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<Option<R>> x=.some(.some(R{.n=65}));return 0;}', 'A', 0, 0),
        ('resource-tag-field', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{Option<S> s;cleanup{mark(82);}}fn i32 main(){let R r=R{.s=.some(S{.n=65})};return 0;}', 'RA', 0, 0),
        ('result-ok', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<R,bool> make(){return .ok(R{.n=65});}fn i32 main(){return switch(make()){case .ok(r):{r.n}case .err(_):{0}};}', 'A', 65, 0),
        ('result-err-resource', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,R> make(){return .err(R{.n=69});}fn Result<i32,R> f(){let R a=R{.n=65};let i32 n=make()?;return .ok(n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(e):{e.n}};}', 'AE', 69, 0),
        ('replace-question-ok', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<R,bool> make(){mark(88);return .ok(R{.n=66});}fn Result<i32,bool> f(){var R old=R{.n=65};old=make()?;return .ok(old.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XAB', 66, 0),
        ('replace-question-err', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<R,bool> make(){mark(88);return .err(true);}fn Result<i32,bool> f(){var R old=R{.n=65};old=make()?;return .ok(old.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XA', 1, 0),
        ('tagged-if-return', 'resource R{i32 n;cleanup{mark(this.n);}}fn Option<R> f(bool yes){return if(yes){.some(R{.n=65})}else{.none};}fn i32 main(){let Option<R> a=f(true);let Option<R> b=f(false);return 0;}', 'A', 0, 0),
    ],
    26: [
        ('partial-factory', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S s;i32 n;cleanup{mark(82);}}fn S make(){return S{.n=65};}fn Result<i32,bool> bad(){return .err(true);}fn Result<i32,bool> f(){let R r=R{.s=make(),.n=bad()?};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'A', 1, 0),
        ('nested-success', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S a;S b;cleanup{mark(82);}}fn i32 main(){let R r=R{.a=S{.n=65},.b=S{.n=66}};return 0;}', 'RBA', 0, 0),
        ('partial-middle', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S a;S b;i32 n;cleanup{mark(82);}}fn Result<i32,bool> bad(){mark(88);return .err(true);}fn Result<i32,bool> f(){let S outer=S{.n=79};let R r=R{.b=S{.n=66},.a=S{.n=65},.n=bad()?};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XABO', 1, 0),
        ('partial-first', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{i32 n;S a;cleanup{mark(82);}}fn Result<i32,bool> bad(){return .err(true);}fn Result<i32,bool> f(){let R r=R{.n=bad()?,.a=S{.n=65}};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', '', 1, 0),
        ('partial-nested', 'resource S{i32 n;cleanup{mark(this.n);}}resource Inner{S s;cleanup{mark(73);}}resource R{Inner a;i32 n;cleanup{mark(82);}}fn Result<i32,bool> bad(){return .err(true);}fn Result<i32,bool> f(){let R r=R{.a=Inner{.s=S{.n=65}},.n=bad()?};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'IA', 1, 0),
        ('later-argument-fails', 'resource S{i32 n;cleanup{mark(this.n);}}fn void take(S a,S b,i32 n){mark(84);}fn Result<i32,bool> bad(){mark(88);return .err(true);}fn Result<i32,bool> f(){take(S{.n=65},S{.n=66},bad()?);return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XBA', 1, 0),
        ('nested-move', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S a;cleanup{mark(82);}}fn R make(){return R{.a=S{.n=65}};}fn i32 main(){let R a=make();let R b=a;return 0;}', 'RA', 0, 0),
        ('temporary-expression', 'resource S{i32 n;cleanup{mark(this.n);}}fn i32 main(){discard S{.n=65};mark(88);return 0;}', 'AX', 0, 0),
    ],
    25: [
        ('overwrite-live', 'resource R{i32 n;cleanup{mark(this.n);}}fn R make(){mark(88);return R{.n=66};}fn i32 main(){var R r=R{.n=65};r=make();mark(89);return r.n;}', 'XAYB', 66, 0),
        ('overwrite-donor', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){var R r=R{.n=65};let R donor=R{.n=66};r=donor;return 0;}', 'AB', 0, 0),
        ('overwrite-self', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){var R r=R{.n=65};r=r;return 0;}', 'A', 0, 0),
        ('overwrite-dead', 'resource R{i32 n;cleanup{mark(this.n);}}fn void take(R r){mark(88);}fn i32 main(){var R r=R{.n=65};take(r);r=R{.n=66};return 0;}', 'XAB', 0, 0),
        ('overwrite-error', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> next(){mark(88);return .err(true);}fn Result<i32,bool> f(){var R r=R{.n=65};r=R{.n=next()?};return .ok(r.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XA', 1, 0),
        ('overwrite-success', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> next(){mark(88);return .ok(66);}fn Result<i32,bool> f(){var R r=R{.n=65};r=R{.n=next()?};return .ok(r.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XAB', 66, 0),
    ],
    24: [
        ('loop-fallthrough', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};var i32 n=0;while(n<3){let R r=R{.n=65};n+=1;}return 0;}', 'AAAO', 0, 0),
        ('loop-continue-break', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};var i32 n=0;while(n<3){let R r=R{.n=65};n+=1;{let R b=R{.n=66};if(n==1){continue;}else{break;};};}mark(88);return 0;}', 'BABAXO', 0, 0),
        ('nearest-loop', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};var i32 n=0;while(n<2){let R a=R{.n=65};while(true){let R b=R{.n=66};break;}mark(88);n+=1;}return 0;}', 'BXABXAO', 0, 0),
        ('for-post-order', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){for(var i32 n=0;n<2;n+=1){let R a=R{.n=65+n};continue;}return 0;}', 'AB', 0, 0),
        ('loop-return', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};while(true){let R a=R{.n=65};return 7;}return 0;}', 'AO', 7, 0),
        ('loop-zero', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};while(false){let R a=R{.n=65};break;}return 0;}', 'O', 0, 0),
    ]
}
REJECTIONS = {
    22: [
        ('break-live', 'resource R{cleanup{}}fn i32 main(){while(true){let R r=R{};break;}return 0;}'),
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){{var R r=R{};r=R{};};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){{let R r=R{.s=S{}};};return 0;}'),
    ],
    23: [
        ('break-live', 'resource R{cleanup{}}fn i32 main(){while(true){let R r=R{};break;}return 0;}'),
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){var R r=R{};r=R{};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}'),
    ],
    27: [
        ('partial-field-move', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn void take(S s){}fn i32 main(){let R r=R{.s=S{}};take(r.s);return 0;}'),
        ('tag-use-after-move', 'resource R{cleanup{}}fn void take(Option<R> r){}fn i32 main(){let Option<R> r=.some(R{});take(r);take(r);return 0;}'),
    ],
    26: [
        ('owned-tag', 'resource S{cleanup{}}fn i32 main(){let Option<S> s=.some(S{});return 0;}'),
        ('field-transfer', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn void take(S s){}fn i32 main(){let R r=R{.s=S{}};take(r.s);return 0;}'),
    ],
    25: [
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}'),
        ('maybe-live', 'resource R{cleanup{}}fn void take(R r){}fn void f(bool yes){var R r=R{};if(yes){take(r);};r=R{};}fn i32 main(){f(true);return 0;}'),
    ],
    24: [
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){var R r=R{};r=R{};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}'),
    ]
}

def sha(path): return hashlib.sha256(path.read_bytes()).hexdigest()
def save(path, value): path.write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n', encoding='utf-8', newline='\n')

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--unit', type=int, required=True)
    ap.add_argument('--images', nargs=2, required=True)
    ap.add_argument('--native-runner', required=True)
    ap.add_argument('--receipt', required=True)
    args = ap.parse_args()
    images = [Path(p).resolve() for p in args.images]
    runner = Path(args.native_runner).resolve()
    receipt = Path(args.receipt).resolve(); receipt.parent.mkdir(parents=True, exist_ok=True)
    rows, fixtures, jobs, native, units = [], [], [], [], []
    passed = False
    def run(argv, expected=0, timeout=120):
        argv = list(map(str, argv))
        p = subprocess.run(argv, cwd=ROOT, capture_output=True, encoding='utf-8', timeout=timeout)
        row = dict(argv=argv, cwd=str(ROOT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr)
        rows.append(row); save(receipt.parent/'raw.json', rows)
        assert p.returncode == expected, row
        return row
    try:
        for name, source, trace, value, trap in CASES[args.unit]:
            path = receipt.parent/(name+'.fy'); path.write_text(PREFIX+source, encoding='utf-8', newline='\n')
            digest = sha(path); outputs = []
            fixtures.append(dict(name=name, inputHex=path.read_bytes().hex(), sha256=digest, expectedTrace=trace, expectedValue=value, expectedTrap=trap))
            for image in images:
                for mode in ('--check-types-file', '--hir-file', '--mir-file', '--c-file'):
                    row = run([image, mode, path]); assert row['stdout'] and not row['stderr'], row
                    if mode == '--c-file': outputs.append(row['stdout'])
                assert sha(path) == digest
            assert outputs[0] == outputs[1], name
            plain = receipt.parent/(name+'.c'); plain.write_text(outputs[0], encoding='utf-8', newline='\n')
            signature = 'void fy_fn_1(int32_t fy_l_1){\n'
            assert outputs[0].count(signature) == 1, name
            # Observation only: record entry to the existing emitted mark body.
            # Neither cleanup registration nor emitted control flow is replaced.
            code = outputs[0].replace(signature, signature+'    (void)fputc((unsigned char)fy_l_1,stdout);(void)fflush(stdout);\n', 1)
            wrapper = '#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n'
            observer = 'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t result=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",result);return 0;}\n'
            observed = receipt.parent/(name+'.observed.c'); observed.write_text(wrapper+code+observer, encoding='utf-8', newline='\n')
            jobs.append(dict(name=name, source=str(observed), observe=not trap))
            units.append(dict(name=name, emittedSHA256=sha(plain), observedSHA256=sha(observed)))
        for name, source in REJECTIONS.get(args.unit, []):
            path = receipt.parent/(name+'.fy'); path.write_text(PREFIX+source, encoding='utf-8', newline='\n')
            fixtures.append(dict(name=name, inputHex=path.read_bytes().hex(), sha256=sha(path), rejectedAt='MIR/C'))
            for image in images:
                for mode in ('--mir-file', '--c-file'):
                    row = run([image, mode, path], 2); assert not row['stdout'] and row['stderr'], row
        request = receipt.parent/'native-request.json'; report = receipt.parent/'native.json'
        save(request, dict(schema='fairyc.candidate-c-native-request/v1', output=str(receipt.parent/'native'), jobs=jobs))
        run([runner, request, report], timeout=300)
        native = json.loads(report.read_text(encoding='utf-8'))
        expected = {name: (trace, value, trap) for name, _, trace, value, trap in CASES[args.unit]}
        assert len(native) == 2*len(jobs)
        for row in native:
            trace, value, trap = expected[row['name']]
            assert not row.get('error') and row['trap'] == trap and row['stdout'] == trace, row
            if not trap: assert row['exit'] == 0 and row['value'] == value, row
        passed = True
    finally:
        save(receipt, dict(schema='fairyc.resource-lifetime-native/v1', unit=args.unit, passed=passed,
            images=[dict(path=str(p), sha256=sha(p)) for p in images], nativeRunner=dict(path=str(runner), sha256=sha(runner)),
            fixtures=fixtures, units=units, cli=rows, native=native,
            sources=[dict(path=str(p.relative_to(REPO)), sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],
            observer=dict(path='examples/lexer_candidate/verify_resource_lifetime.py', sha256=sha(Path(__file__))),
            lock=dict(sha256=sha(ROOT/'fairy.lock'), packages=json.loads((ROOT/'fairy.lock').read_text())['packages'])))
    print(f'PASS unit {args.unit}: {len(fixtures)} fixtures, {len(native)} native runs')

if __name__ == '__main__': main()
