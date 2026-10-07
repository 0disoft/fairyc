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
    ]
}
REJECTIONS = {
    22: [
        ('return-live', 'resource R{cleanup{}}fn i32 main(){let R r=R{};return 0;}'),
        ('break-live', 'resource R{cleanup{}}fn i32 main(){while(true){let R r=R{};break;}return 0;}'),
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){{var R r=R{};r=R{};};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){{let R r=R{.s=S{}};};return 0;}'),
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
