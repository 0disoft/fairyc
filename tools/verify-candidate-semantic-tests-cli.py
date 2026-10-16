"""Compare default candidate test execution with Go, without a native driver."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ('schema = "fairyc.manifest/v1"\n[package]\nuid = "probe.semantic-tests"\n'
            'name = "semantic-tests"\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n')


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def snapshot(home):
    return {p.relative_to(home).as_posix(): (digest(p), p.stat().st_mtime_ns)
            for p in home.rglob('*') if p.is_file()}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--profile', choices=['development', 'release'])
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    build = json.loads(args.build_receipt.read_text(encoding='utf-8'))
    assert build['passed'] and not build['goFrontendInvoked'] and not build.get('commandsOnly') and not build.get('compilerOnly')
    for row in build['sources']:
        assert digest(ROOT / row['path']) == row['sha256'], row['path']
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    reference = args.reference.resolve()
    rows = []

    def run(image, argv, home):
        start = time.monotonic()
        p = subprocess.run([str(image), *argv], cwd=home, input=b'', capture_output=True, timeout=120)
        return dict(exit=p.returncode, stdout=p.stdout.decode('utf-8'),
                    stderr=p.stderr.decode('utf-8'), seconds=time.monotonic() - start)

    for image in build['images']:
        profile = image['profile']
        if args.profile and profile != args.profile:
            continue
        original = Path(image['path'])
        assert digest(original) == image['sha256']
        # Only the compiler image is copied: default tests must work without
        # the native build companion or generated test executables.
        executable = output / profile / 'compiler' / 'fairyc.exe'
        executable.parent.mkdir(parents=True)
        shutil.copyfile(original, executable)
        assert list(executable.parent.iterdir()) == [executable]
        homes = {}

        def fixture(name, files=None):
            home = output / profile / name
            (home / 'src').mkdir(parents=True)
            (home / 'fairy.toml').write_text(MANIFEST, encoding='utf-8')
            (home / 'src/main.fy').write_text('module main;\nfn i32 main(){return 0;}\n', encoding='utf-8')
            for relative, text in (files or {}).items():
                p = home / relative
                p.parent.mkdir(parents=True, exist_ok=True)
                p.write_text(text, encoding='utf-8')
            checked = run(reference, ['check'], home)
            assert checked['exit'] == 0, checked
            homes[name] = home
            return name

        def compare(name, home_name, argv=(), code=0, capability=False, limit=None):
            home = homes[home_name]
            before = snapshot(home)
            go = run(reference, ['test', '--locked', *argv], home)
            candidate = run(executable, ['test', '--locked', *argv], home)
            exact = all(go[k] == candidate[k] for k in ('exit', 'stdout', 'stderr'))
            row = dict(profile=profile, id=name, argv=list(argv), go=go, candidate=candidate,
                       exactMatch=exact, expectedExit=code, capabilityBoundary=capability)
            rows.append(row)
            (output / 'observations.json').write_text(json.dumps(rows, indent=2) + '\n', encoding='utf-8')
            assert snapshot(home) == before, 'test wrote workspace files or changed their timestamps'
            assert candidate['exit'] == code, row
            if capability:
                assert not candidate['stdout'] and 'outside the semantic test interpreter subset:' in candidate['stderr'], row
                assert go['exit'] == 0, row
            else:
                assert exact and go['exit'] == code, row
            if limit:
                assert limit in candidate['stderr'] and not candidate['stdout'], row

        suite = fixture('suite', {
            'src/math.fy': 'module math;pub fn T id<T>(T value){return value;}pub fn i32 add(i32 a,i32 b){return a+b;}\n',
            'tests/basic.fy': '''module tests::basic;
import math as math;
fn i32 factorial(i32 n){return if(n==0){1}else{n*factorial(n-1)};}
fn bool explode(){assert(false);return true;}
fn void frozen(i32 n){assert(n==7);}
@test fn void calls_and_control(){
    var i32 sum=0;
    for(var i32 i=0;i<8;i+=1){if(i==2){continue;}if(i==6){break;}sum+=i;}
    assert(sum==13 && factorial(5)==120 && math::add(20,22)==42);
    assert(math::id<i32>(42)==42 && (false && explode())==false && (true || explode()));
    var i32 j=3;while(j>0){j-=1;}assert(j==0);
    let i32 k={let i32 z=8;z+1};assert(k==9);
}
@test fn void deferred_capture(){var i32 n=7;defer frozen(n);n=99;assert(n==99);}
@test fn void scalar_boundaries(){
    let i32 low=-2147483648;let i64 wide=low as i64;assert(wide== -2147483648);
    let u8 byte=255;let usize size=byte as usize;assert(size==255);
    let usize top=18446744073709551615;assert(top/3==6148914691236517205 && top>9223372036854775807);
    assert((-1<<1)== -2 && (-4>>1)== -2 && (~0)== -1);
}
''',
            'tests/unit/more.fy': 'module tests::unit::more;@test fn void pass(){assert(true);}\n',
        })
        compare('plain', suite)
        compare('json', suite, ['--json'])
        compare('locations', suite, ['--locations'])
        compare('json-locations', suite, ['--json', '--locations'])
        compare('workspace-json', suite, ['--workspace', '--json'])
        compare('package-json', suite, ['--package', 'probe.semantic-tests', '--json'])
        compare('file', suite, ['--file', 'tests/unit/more.fy', '--json'])
        compare('directory', suite, ['--dir', 'tests/unit', '--locations'])
        compare('exact', suite, ['--exact', 'tests::unit::more::pass', '--exact', 'tests::basic::scalar_boundaries', '--json'])
        compare('repeated-exact', suite, ['--exact', 'tests::unit::more::pass', '--exact', 'tests::unit::more::pass'], 2)

        workspace = output / profile / 'workspace'
        shutil.copytree(ROOT / 'examples/workspace_quickstart', workspace,
                        ignore=shutil.ignore_patterns('build', 'fairy.lock'))
        assert run(reference, ['check'], workspace)['exit'] == 0
        homes['workspace'] = workspace
        compare('dependency-call', 'workspace')
        compare('member-execution', 'workspace', ['--package', 'example.workspace-quickstart.answer', '--json'])
        compare('workspace-execution', 'workspace', ['--workspace', '--json', '--locations'])
        compare('workspace-member-file', 'workspace', ['--workspace', '--file', 'libs/answer/tests/answer.fy', '--locations'])

        traps = fixture('traps', {'tests/traps.fy': '''module tests::traps;
@test fn void a_assert(){assert(false);}
@test fn void b_add(){let i32 n=2147483647;discard n+1;}
@test fn void c_division(){let i32 n=0;discard 1/n;}
@test fn void d_division_overflow(){let i32 n=-2147483648;discard n/ -1;}
@test fn void e_shift(){let i32 n=32;discard 1<<n;}
@test fn void f_u8(){let u8 n=255;discard n+1;}
@test fn void g_i64(){let i64 n=9223372036854775807;discard n*2;}
@test fn void h_usize(){let usize n=18446744073709551615;discard n+1;}
@test fn void z_after_traps(){assert(true);}
'''})
        compare('traps-continue', traps, code=5)
        compare('traps-json-locations', traps, ['--json', '--locations'], 5)
        compare('selected-after-traps', traps, ['--exact', 'tests::traps::z_after_traps'])

        empty = fixture('empty')
        compare('zero-tests', empty)
        compare('zero-tests-json-workspace-locations', empty, ['--json', '--workspace', '--locations'])
        compare('missing-selection', suite, ['--exact', 'missing'], 2)
        for flag in ['--release', '--capture-output', '--stdin', '--timeout-seconds']:
            argv = [flag, '1'] if flag == '--timeout-seconds' else [flag]
            compare('native-only-' + flag[2:], suite, argv, 2)

        limits = fixture('limits', {'tests/limits.fy': '''module tests::limits;
fn void recurse(){recurse();}
@test fn void a_before(){assert(true);}
@test fn void depth(){recurse();}
@test fn void steps(){while(true){}}
'''})
        compare('step-limit', limits, ['--exact', 'tests::limits::steps', '--json'], 2, limit='step limit exceeded')
        compare('depth-limit', limits, ['--exact', 'tests::limits::depth'], 2, limit='call-depth limit exceeded')
        compare('no-partial-report', limits, ['--exact', 'tests::limits::a_before', '--exact', 'tests::limits::depth', '--json'], 2)
        isolated = fixture('isolated', {'tests/isolated.fy': '''module tests::isolated;
@test fn void first(){var i32 n=0;while(n<65000){n+=1;}assert(n==65000);}
@test fn void second(){var i32 n=0;while(n<65000){n+=1;}assert(n==65000);}
'''})
        compare('fresh-step-budgets', isolated)

        aggregates = fixture('aggregates', {'tests/aggregates.fy': '''module tests::aggregates;
struct Pair{i32 value;}
@test fn void a_before(){assert(true);}
@test fn void record(){let Pair p=Pair{.value=42};assert(p.value==42);}
@test fn void array(){let i32[2] a=[1,2];assert(a[0]==1);}
'''})
        compare('record', aggregates, ['--exact', 'tests::aggregates::record', '--json'])
        compare('array', aggregates, ['--exact', 'tests::aggregates::array'])
        compare('all-aggregates', aggregates, ['--json'])
        compare('unselected-capability', aggregates, ['--exact', 'tests::aggregates::a_before'])
        copies = fixture('aggregate-copies', {'tests/copies.fy': r'''module tests::copies;
struct Pair{i32 first;i32 second;}
struct Nested{Pair pair;i32[3] items;}
fn Pair mutate(Pair p){var Pair q=p;q.first=9;return q;}
fn Nested update(Nested n){var Nested q=n;q.pair.second=17;q.items[1]=99;return q;}
fn i32[3] array_return(i32[3] items){var i32[3] q=items;q[0]=42;return q;}
@test fn void copies_and_calls(){
 let Pair p=Pair{.second=2,.first=1};var Pair q=mutate(p);q.second=3;
 assert(p.first==1 && p.second==2 && q.first==9 && q.second==3);
 let Nested n=Nested{.items=[4,5,6],.pair=p};let Nested changed=update(n);
 assert(n.pair.second==2 && n.items[1]==5 && changed.pair.second==17 && changed.items[1]==99);
 let i32[3] a=[7;3];let i32[3] b=array_return(a);assert(a[0]==7 && b[0]==42 && a.len==3);
}
@test fn void nested_assignments(){
 var Nested n=Nested{.pair=Pair{.first=1,.second=2},.items=[3,4,5]};let Nested before=n;
 n.pair.first+=8;n.items[1]*=3;let i32 copied=n.items[1];n.items[2]=copied;
 assert(n.pair.first==9 && n.items[1]==12 && n.items[2]==12 && before.pair.first==1 && before.items[1]==4);
 var Pair[2] pairs=[Pair{.first=1,.second=2};2];pairs[0].first=77;assert(pairs[1].first==1);
}
@test fn void aggregate_control_flow(){
 let Pair p=if(true){Pair{.first=1,.second=2}}else{Pair{.first=3,.second=4}};
 let Pair q={let Pair x=p;x};assert(q.first==1 && q.second==2);
 var i32[3] a=[0;3];var usize i=0;while(i<a.len){a[i]=7;i+=1;}assert(a[0]==7 && a[2]==7);
}
'''})
        compare('aggregate-copies-json', copies, ['--json', '--locations'])
        compare('aggregate-copies-file', copies, ['--file', 'tests/copies.fy'])
        bounds = fixture('aggregate-bounds', {'tests/bounds.fy': '''module tests::bounds;
@test fn void a_load(){let i32[2] a=[1,2];let usize i=2;discard a[i];}
@test fn void b_store(){var i32[2] a=[1,2];let usize i=2;a[i]=7;}
@test fn void c_compound(){var i32[2] a=[1,2];let usize i=2;a[i]+=7;}
@test fn void d_after(){assert(true);}
'''})
        compare('aggregate-bounds-json', bounds, ['--json'], 5)
        compare('aggregate-bounds-text', bounds, code=5)
        unsupported = fixture('unsupported', {'tests/resources.fy': '''module tests::resources;
resource R{i32 value;cleanup{}}
@test fn void a_before(){assert(true);}
@test fn void owning_resource(){let R r=R{.value=42};assert(r.value==42);}
'''})
        compare('resource-capability', unsupported, ['--json'], 2, capability=True)
        bad = fixture('invalid', {'tests/bad.fy': 'module tests::bad;@test fn i32 bad(){return 0;}\n'})
        compare('invalid-test-declaration', bad, code=3)
        # The compiler directory also stays free of native tooling/artifacts.
        assert list(executable.parent.iterdir()) == [executable]

    assert rows
    for row in build['sources']:
        assert digest(ROOT / row['path']) == row['sha256'], row['path']
    receipt = dict(schema='fairyc.candidate-semantic-tests-cli/v1', passed=True,
                   buildReceiptSHA256=digest(args.build_receipt), verifierSHA256=digest(Path(__file__)),
                   referenceSHA256=digest(reference), comparisons=rows,
                   exactMatches=sum(r['exactMatch'] for r in rows), cases=len(rows),
                   nativeDriverPresent=False, lockedWorkspaceFilesUnchanged=True)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print('semantic test CLI passed:', len(rows), 'comparisons;', receipt['exactMatches'], 'exact matches', flush=True)


if __name__ == '__main__':
    main()
