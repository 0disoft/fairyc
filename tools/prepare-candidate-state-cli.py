"""Prepare, but do not build, an isolated authored Windows state CLI workspace.

Example: python -B tools/prepare-candidate-state-cli.py --output .cache/state-cli-01
         --bootstrap C:/path/to/fairyc.exe
The manifest records alternative low/optimized build commands. Run exactly the
selected command from its recorded cwd; both use build/fairyc-state.exe. No lock,
SDK source overlay, Go executor, source rewrite, or native build is performed.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import stat
import tomllib

ROOT = Path(__file__).resolve().parents[1]
TARGET = 'x86_64-windows-msvc'
PROFILES = ('low', 'optimized')
PACK_NAMES = {'runtime': 'fairyc-state-graph-runtime.pack',
              'frame': 'fairyc-state-graph-frame.pack'}


def no_links(path):
    """Check lexical ancestors before resolving: junctions count as links."""
    path = Path(path).absolute()
    for part in reversed((path, *path.parents)):
        try:
            info = part.lstat()
        except FileNotFoundError:
            continue
        if stat.S_ISLNK(info.st_mode) or getattr(info, 'st_file_attributes', 0) & 0x400:
            raise ValueError(f'links/reparse points are forbidden: {part}')
    return path


def read_file(path):
    path = no_links(path)
    before = path.stat()
    if not stat.S_ISREG(before.st_mode):
        raise ValueError(f'expected authored regular file: {path}')
    raw = path.read_bytes()
    after = path.stat()
    no_links(path)
    if (before.st_ino, before.st_size, before.st_mtime_ns) != (
            after.st_ino, after.st_size, after.st_mtime_ns):
        raise ValueError(f'input changed while reading: {path}')
    return raw


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def inventory(root):
    """Only authored src, manifest, entry and explicit packaging authorities."""
    source = root / 'examples/lexer_candidate'
    paths = [source / 'fairy.toml', root / 'tools/dev/fairyc-state/main.fy']
    no_links(source / 'src')
    for parent, directories, files in os.walk(source / 'src', followlinks=False):
        for name in directories:
            no_links(Path(parent) / name)
        for name in sorted(files):
            paths.append(Path(parent) / name)
    for kind in PACK_NAMES:
        paths.append(root / f'tools/package-candidate-state-graph-{kind}.py')
    paths.extend(root / 'internal/cstdlib' / name for name in (
        'stage0_contract_state_overlay.h', 'stage0_contract_state_overlay.c',
        'stage0_contract_state_graph.h', 'stage0_contract_state_graph.c',
        'stage0_contract_state_graph_key.c'))
    paths.append(root / 'internal/contractstatenative/graph_native_c.go')
    return {p.relative_to(root).as_posix(): read_file(p) for p in sorted(paths)}


def packs(root, inputs):
    result = {}
    for kind, leaf in PACK_NAMES.items():
        relative = f'tools/package-candidate-state-graph-{kind}.py'
        path = root / relative
        spec = importlib.util.spec_from_file_location(f'state_cli_pack_{kind}', path)
        module = importlib.util.module_from_spec(spec)
        # Execute the inventoried authored bytes, never cached .pyc code. The
        # importlib-created module supplies the original __file__ for ROOT.
        exec(compile(inputs[relative], str(path), 'exec'), module.__dict__)
        data, pin = module.pack_and_pin()
        pin_name = ('module_state_graph_native_asset_pin.fy' if kind == 'runtime'
                    else 'module_state_graph_frame_pin.fy')
        pin_raw = inputs[f'examples/lexer_candidate/src/{pin_name}']
        # Match the owning packagers' existing pin comparison policies.
        actual_pin = pin_raw.decode('utf-8').replace('\r\n', '\n') if kind == 'runtime' else pin_raw
        expected_pin = pin if kind == 'runtime' else pin.encode('utf-8')
        if actual_pin != expected_pin:
            raise ValueError(f'{kind} pin drift; explicit separate pin review required')
        result[leaf] = data
    return result


def fresh_directory(root, output):
    raw = Path(output)
    if '..' in raw.parts:
        raise ValueError('output must not contain parent traversal')
    output = no_links(raw)
    cache = no_links(root / '.cache')
    if output == cache or not output.is_relative_to(cache):
        raise ValueError('output must be a new directory strictly below repository .cache')
    # Never recursively delete, overwrite, or mix different source snapshots.
    if output.exists():
        raise FileExistsError(f'output already exists: {output}')
    for parent in reversed((output.parent, *output.parent.parents)):
        if parent == root or parent.is_relative_to(root):
            if not parent.exists():
                parent.mkdir()
            no_links(parent)
            if not parent.is_dir():
                raise ValueError(f'output ancestor is not a directory: {parent}')
    output.mkdir(exist_ok=False)
    return output


def write_new(path, raw):
    no_links(path)
    with path.open('xb') as stream:
        stream.write(raw)


def prepare(output, bootstrap, *, root=ROOT):
    root = no_links(root)
    bootstrap = no_links(bootstrap)
    bootstrap_raw = read_file(bootstrap)
    inputs = inventory(root)
    config = tomllib.loads(inputs['examples/lexer_candidate/fairy.toml'].decode('utf-8'))
    if config.get('schema') != 'fairyc.manifest/v1' or config.get('package', {}).get('source') != 'src':
        raise ValueError('expected current authored manifest with package.source = src')
    if set(config) - {'schema', 'package'}:
        raise ValueError('manifest now has workspace/dependencies; review isolation before copying')
    if not any(p.startswith('examples/lexer_candidate/src/') for p in inputs):
        raise ValueError('missing authored src tree')
    packaged = packs(root, inputs)
    output = fresh_directory(root, output)
    (output / 'src').mkdir()
    (output / 'build').mkdir()
    copied = []
    mappings = [('examples/lexer_candidate/fairy.toml', 'fairy.toml')]
    for source in inputs:
        prefix = 'examples/lexer_candidate/src/'
        if source.startswith(prefix) and source != prefix + 'main.fy':
            mappings.append((source, 'src/' + source[len(prefix):]))
    mappings.append(('tools/dev/fairyc-state/main.fy', 'src/main.fy'))
    for source, destination in mappings:
        path = output / destination
        path.parent.mkdir(parents=True, exist_ok=True)
        write_new(path, inputs[source])
        copied.append({'source': source, 'destination': destination,
                       'bytes': len(inputs[source]), 'sha256': digest(inputs[source])})
    pack_rows = []
    for leaf, data in packaged.items():
        write_new(output / 'build' / leaf, data)
        pack_rows.append({'path': 'build/' + leaf, 'bytes': len(data), 'sha256': digest(data)})
    if inventory(root) != inputs or read_file(bootstrap) != bootstrap_raw:
        raise ValueError('authored inputs changed during preparation; leave output for inspection, do not reuse')
    manifest = {
        'schema': 'fairyc.prepared-state-cli/v1', 'status': 'prepared-not-built',
        'repository': str(root), 'workspace': str(output), 'target': TARGET,
        'profiles': list(PROFILES), 'profilesAreAlternatives': True,
        'entrySource': 'tools/dev/fairyc-state/main.fy', 'entryDestination': 'src/main.fy',
        'sourceRewrites': False, 'lockCopied': False,
        'stdlib': {'selection': 'bootstrap embedded SDK; no manifest/source overlay',
                   'nativeBuildVerified': False},
        'snapshot': 'Exact bytes read, copied and reread unchanged during preparation; not a commit claim',
        'inputs': [{'path': p, 'bytes': len(raw), 'sha256': digest(raw)} for p, raw in inputs.items()],
        'copiedFiles': copied, 'packs': pack_rows,
        'bootstrap': {'path': str(bootstrap), 'bytes': len(bootstrap_raw), 'sha256': digest(bootstrap_raw)},
        'buildCommands': [{'profile': profile, 'cwd': str(output), 'executed': False,
                           'argv': [str(bootstrap), 'experimental', 'build-workspace',
                                    '--target', TARGET, '--profile', profile,
                                    '--output', 'build/fairyc-state.exe']}
                          for profile in PROFILES],
        'buildPolicy': 'Choose one profile. No --locked on this fresh lock-free workspace. Native integration is separate.',
    }
    write_new(output / 'prepare-manifest.json', (json.dumps(manifest, indent=2) + '\n').encode('utf-8'))
    return manifest


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--bootstrap', type=Path, required=True,
                        help='explicit compiler executable, hashed but never executed')
    args = parser.parse_args(argv)
    try:
        result = prepare(args.output, args.bootstrap)
    except (OSError, ValueError, KeyError) as error:
        parser.error(str(error))
    print(Path(result['workspace']) / 'prepare-manifest.json')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
