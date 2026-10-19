"""Exercise the authored companion, distribution failures and rooted publication."""
import argparse
import ctypes
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--driver-receipt', required=True, type=Path)
    parser.add_argument('--packet', required=True, type=Path, help='SDK fixture printing built by Fairy and exiting 42; include .compile-error and .link-error packets')
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    assert __debug__, 'run without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    build = json.loads(args.driver_receipt.read_text(encoding='utf-8'))
    assert build['passed'] and not build['runtimeGoBuilderRequired']
    sources = build['sources'] + build['runtimeSources']
    assert all(sha(ROOT / row['path']) == row['sha256'] for row in sources)
    bound = [args.driver_receipt.resolve(), Path(__file__).resolve()]
    bound += [Path(str(args.packet.resolve())+suffix) for suffix in ('', '.compile-error', '.link-error')]
    inputs = [dict(path=str(p), sha256=sha(p)) for p in bound]
    raw = ctypes.create_unicode_buffer(32768)
    assert ctypes.windll.kernel32.GetTempPathW(32768, raw) > 0
    temp = Path(raw.value)
    snapshots = lambda: {p.name for p in temp.glob('fairyc-native-*')}
    rows = []

    def run(argv, cwd, code, stdout=None, contains=None):
        before = snapshots()
        result = subprocess.run(list(map(str, argv)), cwd=cwd, capture_output=True, timeout=180)
        row = dict(argv=list(map(str, argv)), exit=result.returncode,
                   stdout=result.stdout.decode(errors='replace'), stderr=result.stderr.decode(errors='replace'))
        rows.append(row)
        (output / 'observations.json').write_text(json.dumps(rows, indent=2)+'\n', encoding='utf-8')
        assert result.returncode == code, row
        assert stdout is None or result.stdout == stdout, row
        assert contains is None or contains in result.stderr, row
        assert snapshots() == before, 'owned scratch leaked'
        return result

    products = {}
    for image in build['images']:
        driver = Path(image['path'])
        pack = Path(image['assetPath'])
        assert sha(driver) == image['sha256'] and sha(pack) == image['assetSHA256']
        home = output / image['profile'] / '한글 distribution'
        home.mkdir(parents=True)
        installed = home / driver.name
        shutil.copyfile(driver, installed)
        asset = home / pack.name
        shutil.copyfile(pack, asset)
        protected = home / 'build/product.exe'
        argv = [installed, '--packet-input', args.packet.resolve(), '--output-root', home,
                '--output', 'build/product.exe', '--compile-timeout', '1m']
        for profile, flags in [('development', []), ('release', ['--release'])]:
            built = run([*argv, *flags], home, 0)
            assert not built.stderr
            run([protected], home, 42, stdout=b'built by Fairy\n')
            products[(image['profile'], profile)] = sha(protected)
        preserved = protected.read_bytes()
        for suffix, label in [('.compile-error', b'compile MSVC target C'), ('.link-error', b'link MSVC target executable')]:
            failed = list(argv)
            failed[2] = Path(str(args.packet.resolve())+suffix)
            run(failed, home, 1, stdout=b'', contains=label)
            assert protected.read_bytes() == preserved
        # An authentic pack in cwd cannot substitute for the companion asset.
        cwd = home / 'cwd'
        cwd.mkdir()
        shutil.copyfile(asset, cwd / asset.name)
        valid = asset.read_bytes()
        asset.unlink()
        run(argv, cwd, 1, stdout=b'', contains=b'runtime asset pack is missing or unreadable')
        assert protected.read_bytes() == preserved
        asset.write_bytes(bytes([valid[0] ^ 1])+valid[1:])
        run(argv, cwd, 1, stdout=b'', contains=b'runtime asset pack does not match compiler pin')
        assert protected.read_bytes() == preserved
        asset.write_bytes(valid)
        # Opening the child parent must reject directory aliases before staging.
        outside = home / 'outside'
        outside.mkdir()
        marker = outside / 'product.exe'
        marker.write_bytes(b'outside image')
        (home / 'alias').symlink_to(outside, target_is_directory=True)
        escaped = list(argv)
        escaped[6] = 'alias/product.exe'
        run(escaped, home, 1, stdout=b'', contains=b'publish candidate image failed')
        assert marker.read_bytes() == b'outside image'
        escaped[6] = '../outside.exe'
        run(escaped, home, 1, stdout=b'', contains=b'publish candidate image failed')
        assert not (home.parent / 'outside.exe').exists()
        assert not list(home.rglob('.fairyc-candidate-*.tmp'))
        assert sha(driver) == image['sha256'] and sha(pack) == image['assetSHA256']
    for profile in ('development', 'release'):
        assert products[('development', profile)] == products[('release', profile)]
    assert all(sha(ROOT / row['path']) == row['sha256'] for row in sources)
    assert all(sha(Path(row['path'])) == row['sha256'] for row in inputs)
    receipt = dict(schema='fairyc.candidate-authored-driver-verification/v1', passed=True,
                   inputs=inputs, driverBuild=build, executions=len(rows), rows=rows,
                   products=[dict(driver=a, target=b, sha256=h) for (a, b), h in products.items()],
                   runtimeGoInvoked=False, missingAndTamperedAssetsRejected=True,
                   cwdAssetFallback=False, rootedPublication=True, scratchCleaned=True,
                   linuxExecuted=False, fullCliFixedPointRefreshed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')
    print('Authored driver verification passed:', len(rows), 'executions')


if __name__ == '__main__':
    main()
