"""Compare public candidate loop-control diagnostics with the Go reference."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
CASES = [
    ('outside-break', 'fn i32 main(){break;return 0;}', 'E1012'),
    ('outside-continue', 'fn i32 main(){continue;return 0;}', 'E1012'),
    ('expression-block', 'fn i32 main(){return if(true){{break;0}}else{1};}', 'E1012'),
    ('loop-condition', 'fn i32 main(){while({break;true}){}return 0;}', 'E1012'),
    ('other-function', 'fn i32 valid(){while(true){break;}return 0;}fn i32 main(){continue;return 0;}', 'E1012'),
    ('nested-body', 'fn i32 main(){while(true){if(true){break;}continue;}return 0;}', None),
    ('outer-loop-condition', 'fn i32 main(){while(true){while({break;true}){}break;}return 0;}', None),
    ('classic-for', 'fn i32 main(){for(var usize i=0;i<4;i+=1){continue;break;}return 0;}', None),
    ('byte-for', 'fn void each(ref u8[] data){for(let u8 byte in data){continue;break;}}fn i32 main(){return 0;}', None),
]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    tools = {name: dict(path=str(path.resolve()), sha256=digest(path))
             for name, path in [('candidate', args.candidate), ('reference', args.reference)]}
    source_paths = [ROOT / path for path in [
        'examples/lexer_candidate/src/module_unit_loop_control.fy',
        'examples/lexer_candidate/src/module_unit_value_collect.fy',
        'examples/lexer_candidate/src/project_check_text.fy',
        'tools/verify-candidate-loop-control.py',
    ]]
    sources = [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in source_paths]
    rows = []
    receipt = dict(schema='fairyc.candidate-loop-control/v1', passed=False,
                   tools=tools, sourceInputs=sources, cases=rows, pushed=False)

    def save():
        (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')

    for index, (name, body, code) in enumerate(CASES):
        workspace = output / name
        (workspace / 'src').mkdir(parents=True, exist_ok=True)
        (workspace / 'fairy.toml').write_text(
            'schema = "fairyc.manifest/v1"\n[package]\n'
            f'uid = "fixture.loop.case{index}"\nname = "loop-fixture"\n'
            'version = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n',
            encoding='utf-8', newline='\n')
        source = workspace / 'src/main.fy'
        source.write_text('module main;' + body, encoding='utf-8', newline='\n')
        observations = []
        for engine, tool in tools.items():
            result = subprocess.run([tool['path'], 'check'], cwd=workspace, capture_output=True, timeout=60)
            stderr = result.stderr.decode('utf-8')
            diagnostics = [dict(code=c, start=int(a), end=int(b)) for c, a, b in
                           re.findall(r'(E\d{4}).*?\[bytes (\d+)\.\.(\d+)', stderr)]
            observations.append(dict(engine=engine, exit=result.returncode,
                                     stdout=result.stdout.decode('utf-8'), stderr=stderr, diagnostics=diagnostics))
        row = dict(name=name, sourceSHA256=digest(source), expectedCode=code, observations=observations)
        rows.append(row)
        save()
        candidate, reference = observations
        assert candidate['exit'] == reference['exit'] == (3 if code else 0), row
        assert candidate['diagnostics'][:1] == reference['diagnostics'][:1], row
        if code:
            assert reference['diagnostics'][0]['code'] == code, row
        else:
            assert not candidate['diagnostics'] and not reference['diagnostics'], row
    assert all(digest(Path(item['path'])) == item['sha256'] for item in tools.values())
    assert all(digest(ROOT / item['path']) == item['sha256'] for item in sources)
    receipt['passed'] = True
    save()
    print(len(rows), 'public loop-control cases match the Go reference', flush=True)


if __name__ == '__main__':
    main()
