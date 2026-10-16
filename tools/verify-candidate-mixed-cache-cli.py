"""Verify mixed package reuse through fresh candidate check/build/run processes."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--profile', action='append', choices=['development', 'release'],
                        help='verify only this native compiler profile (default: both)')
    args = parser.parse_args()
    assert __debug__, 'verification requires Python without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    build = json.loads(args.build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    assert {r['profile'] for r in build['images']} == {'development', 'release'}
    reference = args.reference.resolve()
    reference_sha = sha(reference)
    inputs = build['sources'] + [dict(path=p, sha256=sha(ROOT / p)) for p in
                                ['tools/verify-candidate-mixed-cache-cli.py', 'tools/verify-candidate-mixed-cache-fixtures.py']]
    assert all(sha(ROOT / r['path']) == r['sha256'] for r in inputs)
    rows = []

    def run(image, cwd, command, flags=(), expected=0, status=b'', label=''):
        started = time.monotonic()
        result = subprocess.run([str(image), command, *map(str, flags)], cwd=cwd,
                                capture_output=True, timeout=900)
        rows.append(dict(label=label, image=str(image), command=command, flags=list(map(str, flags)),
                         cwd=str(cwd), seconds=time.monotonic()-started, exit=result.returncode,
                         stdout=result.stdout.decode('utf-8'), stderr=result.stderr.decode('utf-8')))
        (output / 'executions.json').write_text(json.dumps(rows, indent=2)+'\n', encoding='utf-8')
        assert result.returncode == expected and result.stderr.startswith(status), rows[-1]
        print('passed:', label, flush=True)
        return result.stdout, result.stderr[len(status):]

    def snapshot(cache):
        return {p.name: sha(p) for p in cache.glob('fyproject0-*.fyc')}

    def projects(cache):
        return {p.name for p in cache.glob('fyproject0-*.fyc') if p.read_bytes()[16:24] == b'FYCKEYS0'}

    selected_images = [p for p in build['images'] if not args.profile or p['profile'] in args.profile]
    assert selected_images
    for selected in selected_images:
        profile = selected['profile']
        image = Path(selected['path'])
        assert sha(image) == selected['sha256']
        assert sha(image.parent / 'fairyc-candidate-build.exe') == selected['driverSHA256']
        base = output / profile / 'fixtures'
        subprocess.run([sys.executable, str(ROOT / 'tools/verify-candidate-mixed-cache-fixtures.py'),
                        '--output', str(base)], cwd=ROOT, check=True, capture_output=True)
        cache = output / profile / 'cache'
        cache.mkdir()
        flags = ['--cache-root', cache, '--cache-status']
        root = base / 'Base'
        run(image, root, 'check', flags, status=b'cache: cold stored\n', label=profile+' cold population')
        assert len(snapshot(cache)) == 4
        assert run(image, root, 'check', ['--locked', *flags], status=b'cache: hit\n', label=profile+' project hit') == run(reference, root, 'check', ['--locked'], label=profile+' Go baseline oracle')
        project_record = next(p for p in cache.glob('fyproject0-*.fyc') if p.read_bytes()[16:24] == b'FYCKEYS0')
        unit_record = next(p for p in cache.glob('fyproject0-*.fyc') if p.read_bytes()[16:24] == b'FYPUNIT0')
        project_record.write_bytes(b'broken')
        unit_record.write_bytes(b'broken')
        run(image, root, 'check', ['--locked', *flags], status=b'cache: mixed\n', label=profile+' corrupt unit rebuilds beside valid units')
        run(image, root, 'check', ['--locked', *flags], status=b'cache: hit\n', label=profile+' repaired project hit')
        helper = root / 'helper/src/helper.fy'
        helper.write_text(helper.read_text(encoding='utf-8').replace('42', '43'), encoding='utf-8')
        run(image, root, 'check', flags, status=b'cache: mixed\n', label=profile+' private dependency edit')
        assert run(image, root, 'run', ['--locked', *flags], 44, b'cache: hit\n', profile+' mixed result executes') == (b'', b'')
        source = root / 'src/main.fy'
        source.write_text(source.read_text(encoding='utf-8').replace('h::get()+o::one()', 'h::get()+o::one()+1'), encoding='utf-8')
        before = projects(cache)
        run(image, root, 'build', flags, status=b'cache: mixed\n', label=profile+' root edit mixed build')
        newest = projects(cache) - before
        assert len(newest) == 1
        assert run(image, root, 'run', ['--locked', *flags], 45, b'cache: hit\n', profile+' mixed build execution') == (b'', b'')
        product = root / 'build/mixed-cache.exe'
        product_sha = sha(product)
        (cache / newest.pop()).write_bytes(b'broken')
        assert run(image, root, 'check', ['--locked', *flags], status=b'cache: units hit\n', label=profile+' corrupt project uses admitted units') == run(reference, root, 'check', ['--locked'], label=profile+' Go recovered oracle')
        retained = snapshot(cache)
        helper.write_text(helper.read_text(encoding='utf-8').replace('return 43', 'return true'), encoding='utf-8')
        assert run(image, root, 'build', flags, 3, b'cache: cold uncached\n', profile+' invalid body') == run(reference, root, 'check', ['--locked'], 3, label=profile+' Go diagnostic oracle')
        assert snapshot(cache) == retained and sha(product) == product_sha
        helper.write_text(helper.read_text(encoding='utf-8').replace('return true', 'return 43'), encoding='utf-8')
        helper.write_text(helper.read_text(encoding='utf-8').replace('pub fn', '@effects(console) pub fn'), encoding='utf-8')
        run(image, root, 'check', flags, status=b'cache: mixed\n', label=profile+' public effect invalidates caller')
        assert run(image, root, 'run', ['--locked', *flags], 45, b'cache: hit\n', profile+' public contract result') == (b'', b'')
        for first, second, expected, label in [('Generic', 'GenericEdit', 43, 'generic body'), ('Vec', 'VecEdit', 44, 'Vec ordinary dependency')]:
            selected_cache = output / profile / first
            selected_cache.mkdir()
            selected_flags = ['--cache-root', selected_cache, '--cache-status']
            run(image, base / first, 'check', selected_flags, status=b'cache: cold stored\n', label=profile+' '+label+' population')
            run(image, base / second, 'run', selected_flags, expected, b'cache: mixed\n', profile+' '+label+' mixed execution')
        assert not list(cache.glob('.stage*'))
        assert sha(image) == selected['sha256']
    assert all(sha(ROOT / r['path']) == r['sha256'] for r in inputs)
    assert sha(reference) == reference_sha
    report = dict(schema='fairyc.candidate-mixed-cache-cli/v0', passed=True,
                  sourceInputs=inputs, buildReceiptSHA256=sha(args.build_receipt),
                  referenceSHA256=reference_sha, images=selected_images, executions=rows,
                  publicApiChanged=False, sdkAbiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print('Mixed package commands passed:', len(rows), 'process observations')


if __name__ == '__main__':
    main()
