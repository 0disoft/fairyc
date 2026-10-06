#!/usr/bin/env python3
"""Fixed Result execution fixtures; product compilation runs entirely in Fairy."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

from verify_results import SCALARS, VALUES, PATTERN_NEGATIVES, sha

ROOT = Path(__file__).resolve().parent
POSITIVES = []
for success in SCALARS:
    for failure in SCALARS:
        # Both alternatives cross a call/return boundary and their differently
        # typed captures must equal the exact scalar boundary values.
        POSITIVES.append((f'pair-{success}-{failure}',
            f'fn Result<{success},{failure}> make(bool ok){{return if(ok){{.ok({VALUES[success]})}}else{{.err({VALUES[failure]})}};}}'
            f'fn i32 read(Result<{success},{failure}> r){{return switch(r){{case .ok(n):{{assert(n=={VALUES[success]});1}}case .err(e):{{assert(e=={VALUES[failure]});2}}}};}}'
            'fn i32 main(){assert(read(make(true))==1);assert(read(make(false))==2);return 0;}', 0))
POSITIVES += [
    ('ok', 'fn i32 main(){let Result<i32,bool> r=.ok(7);return switch(r){case .ok(n):{n}case .err(e):{if(e){1}else{0}}};}', 7),
    ('err', 'fn i32 main(){let Result<i32,bool> r=.err(true);return switch(r){case .err(e):{if(e){7}else{0}}case .ok(n):{n}};}', 7),
    ('copy', 'fn Result<i32,bool> id(Result<i32,bool> r){return r;}fn i32 main(){var Result<i32,bool> r=.ok(7);let saved=id(r);r=.err(false);discard r;return switch(saved){case .ok(n):{n}case .err(_):{0}};}', 7),
    ('early-argument-frozen', 'fn i32 read(Result<i32,bool> r,bool ignored){return switch(r){case .ok(n):{n}case .err(_):{0}};}fn i32 main(){var Result<i32,bool> r=.ok(7);return read(r,{r=.err(true);discard r;false});}', 7),
    ('discard', 'fn i32 main(){let Result<i32,bool> r=.err(true);return switch(r){case .ok(_):{1}case .err(_):{7}};}', 7),
    ('subject-once', 'fn i32 main(){var i32 count=0;var Result<i32,bool> r=.ok(8);return switch({count+=1;r}){case .ok(n):{r=.err(false);discard r;count+=2;n+count}case .err(_):{0}};}', 11),
    ('return-arm-first', 'fn i32 read(Result<i32,bool> r){return switch(r){case .err(e):{return if(e){2}else{3};}case .ok(n):{n}};}fn i32 main(){return read(.ok(7))+read(.err(true))+read(.err(false));}', 12),
    ('return-arm-last', 'fn i32 read(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(_):{return 2;}};}fn i32 main(){return read(.ok(7))+read(.err(false));}', 9),
    ('both-return', 'fn i32 read(Result<i32,bool> r){switch(r){case .ok(n):{return n;}case .err(_):{return 2;}};return 99;}fn i32 main(){return read(.ok(7))+read(.err(true));}', 9),
    ('assert-arm', 'fn i32 main(){let Result<i32,bool> r=.ok(7);let n=switch(r){case .err(_):{assert(false);}case .ok(n):{n}};return n;}', 7),
    ('early-ok', 'fn i32 main(){let Result<i32,bool> r=.ok({return 6;7});return switch(r){case .ok(n):{n}case .err(_):{0}};}', 6),
    ('early-err', 'fn i32 main(){let Result<i32,bool> r=.err({return 6;true});return switch(r){case .ok(n):{n}case .err(_):{0}};}', 6),
    ('loop', 'fn i32 main(){var Result<i32,bool> r=.ok(0);var i32 i=0;while(i<3){r=switch(r){case .ok(n):{.ok(n+1)}case .err(_):{.ok(1)}};i+=1;}return switch(r){case .ok(n):{n}case .err(_):{99}};}', 3),
    ('shadow', 'fn i32 main(){let i32 n=4;let Result<i32,bool> r=.ok(3);return switch(r){case .ok(n):{n}case .err(n):{if(n){1}else{0}}}+n;}', 7),
    ('coexist', 'enum E{Data{i32 n;};}fn i32 main(){let E e=.Data{.n=2};let Option<i32> o=.some(3);let Result<i32,bool> r=.err(true);return switch(e){case .Data{.n=n}:{n}}+switch(o){case .some(n):{n}case .none:{0}}+switch(r){case .ok(n):{n}case .err(e):{if(e){4}else{0}}};}', 9),
    ('ordered-pairs', 'fn i32 main(){let Result<i32,bool> a=.ok(3);let Result<bool,i32> b=.err(4);return switch(a){case .ok(n):{n}case .err(_):{0}}+switch(b){case .ok(_):{0}case .err(n):{n}};}', 7),
]
TRAPS = [
    ('ok-overflow-trap', 'fn i32 main(){let Result<i32,bool> r=.ok(2147483647+1);return switch(r){case .ok(n):{n}case .err(_):{0}};}', 1),
    ('err-division-trap', 'fn i32 main(){let Result<bool,i32> r=.err(1/0);return switch(r){case .ok(_):{0}case .err(n):{n}};}', 2),
    ('payload-assert-trap', 'fn i32 main(){let Result<i32,bool> r=.err({assert(false);true});return switch(r){case .ok(n):{n}case .err(_):{0}};}', 5),
    ('arm-assert-trap', 'fn i32 main(){let Result<i32,bool> r=.err(true);return switch(r){case .ok(n):{n}case .err(_):{assert(false);}};}', 5),
]

def save(path, data):
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2)+'\n', encoding='utf-8', newline='\n')

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--images', nargs='+', required=True)
    parser.add_argument('--native-runner', required=True)
    parser.add_argument('--receipt', required=True)
    parser.add_argument('--propagation', action='store_true')
    parser.add_argument('--must-use', action='store_true')
    parser.add_argument('--nested-tagged', action='store_true')
    parser.add_argument('--struct-payloads', action='store_true')
    parser.add_argument('--tagged-fields', action='store_true')
    parser.add_argument('--tagged-arrays', action='store_true')
    args = parser.parse_args()
    positives, traps, negatives, traces = POSITIVES, TRAPS, PATTERN_NEGATIVES, {}
    if args.propagation:
        from verify_propagation_cases import POSITIVES as positives, TRAPS as traps, NEGATIVES as negatives, TRACES as traces
    if args.must_use:
        from verify_result_handling_cases import POSITIVES as positives, TRAPS as traps, NEGATIVES as negatives
    if args.nested_tagged:
        from verify_nested_tagged_cases import POSITIVES as positives, TRAPS as traps, NEGATIVES as negatives
    if args.struct_payloads:
        from verify_struct_payload_cases import POSITIVES as positives, TRAPS as traps, NEGATIVES as negatives
    if args.tagged_fields:
        from verify_tagged_field_cases import POSITIVES as positives, TRAPS as traps, NEGATIVES as negatives
    if args.tagged_arrays:
        from verify_tagged_array_cases import POSITIVES as positives, TRAPS as traps, NEGATIVES as negatives, TRACES as traces
    composite = args.nested_tagged or args.struct_payloads or args.tagged_fields or args.tagged_arrays
    images = [Path(p).resolve() for p in args.images]
    runner = Path(args.native_runner).resolve()
    receipt = Path(args.receipt).resolve(); receipt.parent.mkdir(parents=True, exist_ok=True)
    rows, fixtures, jobs, native = [], [], [], []
    passed = False
    def run(argv, expected=0, timeout=120):
        argv = list(map(str, argv))
        p = subprocess.run(argv, cwd=ROOT, capture_output=True, encoding='utf-8', timeout=timeout)
        row = dict(argv=argv, cwd=str(ROOT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr)
        rows.append(row); assert p.returncode == expected, row
        return row
    modes = ['--check-types-file', '--hir-file', '--mir-file', '--c-file']
    try:
        for rejected, cases in ((False, positives+traps), (True, negatives)):
            for name, source, expected in cases:
                path = receipt.parent/(name+'.fy'); data = ('module main;'+source).encode('utf-8')
                path.write_bytes(data); digest = sha(path)
                fixtures.append(dict(name=name, inputHex=data.hex(), sha256=digest, rejected=rejected, expected=expected))
                outputs = {mode: [] for mode in modes}
                for image in images:
                    for mode in modes:
                        row = run([image, mode, path], 2 if rejected else 0)
                        if rejected:
                            assert not row['stdout'] and row['stderr'] and expected in row['stderr'], row
                        else:
                            assert row['stdout'] and not row['stderr'], row
                        outputs[mode].append(row['stdout'])
                    assert sha(path) == digest
                assert all(len(set(values)) == 1 for values in outputs.values())
                if rejected: continue
                if args.tagged_arrays: assert outputs['--mir-file'][0].startswith(('fairyc.enum-payload-mir/v1\n','fairyc.enum-mir/v1\n','fairyc.nominal-mir/v1\n','fairyc.array-mir/v1\n'))
                elif args.tagged_fields: assert outputs['--mir-file'][0].startswith(('fairyc.enum-payload-mir/v1\n','fairyc.enum-mir/v1\n'))
                else: assert outputs['--mir-file'][0].startswith('fairyc.enum-payload-mir/v1\n')
                if not composite: assert '\t2621' in outputs['--mir-file'][0] and 'result:' in outputs['--hir-file'][0]
                if args.propagation: assert '\tpropagate\t' in outputs['--hir-file'][0]
                code = outputs['--c-file'][0]
                if composite:
                    assert code.startswith('/* fairyc.candidate-') and ('struct fy_r_' in code or 'struct fy_o_' in code or args.tagged_fields and 'struct fy_s_' in code or args.tagged_arrays and 'struct fy_a_' in code)
                else: assert code.startswith('/* fairyc.candidate-result-c/v1') and 'struct fy_r_' in code
                trap = name in {n for n, _, _ in traps}
                unit = receipt.parent/(name+'.c')
                wrapper = '#define FY_CANDIDATE_NO_MAIN\n#include <inttypes.h>\n#include <stdio.h>\n'
                observer = 'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t result=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",result);return 0;}\n'
                if name in traces:
                    # Measurement only: record actual calls to the emitted mark
                    # function. Product registration, exits and call order stay
                    # unchanged. Check the trace after native entry returns.
                    signature = 'void fy_fn_1(int32_t fy_l_1){'
                    assert code.count(signature) == 1
                    code = code.replace(signature, signature+'\n    if(fy_trace_count>=8) abort();fy_trace[fy_trace_count++]=fy_l_1;\n', 1)
                    wrapper += '#include <stdlib.h>\nstatic int32_t fy_trace[8];static int32_t fy_trace_count;\n'
                    checks = f'if(fy_trace_count!={len(traces[name])}) return 90;'
                    checks += ''.join(f'if(fy_trace[{i}]!={n}) return 91;' for i,n in enumerate(traces[name]))
                    observer = observer.replace('fprintf(stderr,', checks+'fprintf(stderr,', 1)
                unit.write_text(code if trap else wrapper+code+observer, encoding='utf-8', newline='\n')
                jobs.append(dict(name=name, source=str(unit), observe=not trap))
                if name in ('ok', 'err'):
                    old, new = ('0', '1') if name == 'ok' else ('1', '0')
                    make = f'.fy_tag=UINT32_C({old})'; branch = f'.fy_tag == UINT32_C({old})'
                    assert code.count(make) == 1 and code.count(branch) == 1
                    corrupted = code.replace(make, f'.fy_tag=UINT32_C({new})', 1).replace(branch, f'.fy_tag == UINT32_C({new})', 1)
                    fault = receipt.parent/(name+'-inactive.c'); fault.write_text(corrupted, encoding='utf-8', newline='\n')
                    jobs.append(dict(name=name+'-inactive', source=str(fault), observe=False))
        request = receipt.parent/'native-request.json'; report = receipt.parent/'native.json'
        save(request, dict(schema='fairyc.candidate-c-native-request/v1', output=str(receipt.parent/'native'), jobs=jobs))
        run([runner, request, report], timeout=300)
        native = json.loads(report.read_text(encoding='utf-8'))
        assert len(native) == (2 if sys.platform == 'win32' else 4)*len(jobs)
        expected_values = {name: value for name, _, value in positives}
        expected_traps = {name: value for name, _, value in traps} | {'ok-inactive': 7, 'err-inactive': 7}
        for row in native:
            if row['name'] in expected_traps:
                assert row['trap'] == expected_traps[row['name']] and row['exit'] != 0 and not row['stdout'] and not row['stderr'], row
            else:
                expected = expected_values[row['name']]
                assert row['exit'] == 0 and row['trap'] == 0 and row['value'] == expected and not row['stdout'], row
                assert row['stderr'].replace('\r\n','\n') == f'fairyc-candidate-result/v1:{expected}\n', row
            assert sha(row['image']) == row['imageSHA256'] and sha(row['source']) == row['sourceSHA256'], row
        passed = True
    finally:
        save(receipt, dict(schema='fairyc.result-candidate-execution/v1', unit=14 if args.tagged_fields or args.tagged_arrays else 13 if args.struct_payloads else 12 if args.nested_tagged else 11 if args.must_use else 10 if args.propagation else 8, part='arrays-native' if args.tagged_arrays else 'fields' if args.tagged_fields else None, passed=passed,
            images=[dict(path=str(p), sha256=sha(p)) for p in images],
            nativeRunner=dict(path=str(runner), sha256=sha(runner)), cli=rows, native=native, fixtures=fixtures,
            faultInjection=None if args.propagation or args.must_use or composite else dict(kind='test-only C tag and branch corruption in both alternatives', validFairySource=False, expectedTrap=7),
            cleanupObservation=dict(kind='test-only observation of actual generated-C mark calls', expectedTraces=traces),
            sources=[dict(path=str(p.relative_to(ROOT)), sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))]))
    print(f'PASS: Result execution; {len(rows)} CLI observations; {len(native)} native executions')

if __name__ == '__main__': main()
