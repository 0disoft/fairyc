"""Package an already built private Windows developer CLI; never build or run it.

python -B tools/package-candidate-state-cli.py --workspace .cache/state-cli-host-low
    --build-record .cache/state-cli-host-build-low.json --output .cache/state-cli-low.zip
"""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import struct
import zipfile

TOOLS = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('prepare_state_cli', TOOLS / 'prepare-candidate-state-cli.py')
prepare = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prepare)
ROOT = prepare.ROOT
IMAGE = 'build/fairyc-state.exe'
README = b'''# Private Windows state developer CLI

Keep fairyc-state.exe, fairyc-state-graph-runtime.pack and
fairyc-state-graph-frame.pack together in this directory.
From this directory in PowerShell: .\\fairyc-state.exe --help
See LICENSE, metadata.json and SHA256SUMS in this directory.

This is a local-only Windows x64 developer snapshot. Packaging verifies the
recorded native build and exact current source, bootstrap, image and pack bytes.
Source hashes authenticate that snapshot, not a Git commit. Packaging does not
execute the image or establish clean-machine, installed, Linux, hosted or public
release acceptance. The private native graph command needs its existing local
native toolchain; this ZIP does not install that toolchain or dependencies.
'''


def json_bytes(value):
    return (json.dumps(value, indent=2, sort_keys=True) + '\n').encode('utf-8')


def identity(raw):
    return {'bytes': len(raw), 'sha256': prepare.digest(raw)}


def require(actual, expected, label):
    if actual != expected:
        raise ValueError(f'{label} mismatch or stale snapshot')


def cache_path(root, path):
    path = Path(path)
    if '..' in path.parts:
        raise ValueError('parent traversal is forbidden')
    path = prepare.no_links(path)
    cache = prepare.no_links(root / '.cache')
    if path == cache or not path.is_relative_to(cache):
        raise ValueError('path must be strictly below repository .cache')
    return path


def windows_image(raw):
    """Check PE32+ AMD64 executable headers and bounded file-backed sections."""
    if len(raw) < 64 or raw[:2] != b'MZ':
        raise ValueError('image must be a Windows x64 PE executable')
    pe = struct.unpack_from('<I', raw, 0x3c)[0]
    if pe < 64 or pe + 24 > len(raw) or raw[pe:pe + 4] != b'PE\0\0':
        raise ValueError('invalid PE signature/offset')
    machine, count, _, _, _, optional_size, flags = struct.unpack_from('<HHIIIHH', raw, pe + 4)
    optional = pe + 24
    table = optional + optional_size
    if (machine != 0x8664 or not 1 <= count <= 96 or optional_size < 112
            or not flags & 2 or flags & 0x2000 or table + count * 40 > len(raw)):
        raise ValueError('invalid Windows x64 PE executable headers')
    magic = struct.unpack_from('<H', raw, optional)[0]
    entry = struct.unpack_from('<I', raw, optional + 16)[0]
    image_size, headers = struct.unpack_from('<II', raw, optional + 56)
    subsystem = struct.unpack_from('<H', raw, optional + 68)[0]
    directories = struct.unpack_from('<I', raw, optional + 108)[0]
    if (magic != 0x20b or subsystem != 3 or not image_size
            or not table + count * 40 <= headers <= len(raw)
            or 112 + directories * 8 > optional_size):
        raise ValueError('invalid PE32+ console optional header')
    executable_entry = False
    for index in range(count):
        section = table + index * 40
        virtual_size, address, size, offset = struct.unpack_from('<IIII', raw, section + 8)
        characteristics = struct.unpack_from('<I', raw, section + 36)[0]
        if size and (offset < headers or offset + size > len(raw)):
            raise ValueError('PE section outside image file')
        if address + max(virtual_size, size) > image_size:
            raise ValueError('PE section outside mapped image')
        if characteristics & 0x20000000 and address <= entry < address + size:
            executable_entry = True
    if not executable_entry:
        raise ValueError('PE entry point is not in a file-backed executable section')


def validated_payload(root, workspace, record_path):
    manifest_raw = prepare.read_file(workspace / 'prepare-manifest.json')
    record_raw = prepare.read_file(record_path)
    manifest = json.loads(manifest_raw)
    record = json.loads(record_raw)
    for key, expected in {
        'schema': 'fairyc.prepared-state-cli/v1', 'status': 'prepared-not-built',
        'repository': str(root), 'workspace': str(workspace), 'target': prepare.TARGET,
        'profiles': list(prepare.PROFILES), 'profilesAreAlternatives': True,
        'entrySource': 'tools/dev/fairyc-state/main.fy', 'entryDestination': 'src/main.fy',
        'sourceRewrites': False, 'lockCopied': False,
        'stdlib': {'selection': 'bootstrap embedded SDK; no manifest/source overlay',
                   'nativeBuildVerified': False},
    }.items():
        require(manifest[key], expected, f'manifest {key}')
    inputs = prepare.inventory(root)
    require(manifest['inputs'], [{'path': p, **identity(raw)} for p, raw in inputs.items()], 'authored inputs')
    mappings = [('examples/lexer_candidate/fairy.toml', 'fairy.toml')]
    prefix = 'examples/lexer_candidate/src/'
    mappings.extend((p, 'src/' + p[len(prefix):]) for p in inputs
                    if p.startswith(prefix) and p != prefix + 'main.fy')
    mappings.append(('tools/dev/fairyc-state/main.fy', 'src/main.fy'))
    require(manifest['copiedFiles'], [{'source': s, 'destination': d, **identity(inputs[s])}
                                    for s, d in mappings], 'copied inventory')
    for source, destination in mappings:
        require(prepare.read_file(workspace / destination), inputs[source], f'copied {destination}')
    actual_sources = set()
    prepare.no_links(workspace / 'src')
    for parent, directories, files in os.walk(workspace / 'src', followlinks=False):
        for name in directories + files:
            prepare.no_links(Path(parent) / name)
        actual_sources.update((Path(parent) / name).relative_to(workspace).as_posix() for name in files)
    require(actual_sources, {d for _, d in mappings if d.startswith('src/')}, 'copied source tree')
    bootstrap = Path(manifest['bootstrap']['path'])
    if not bootstrap.is_absolute() or '..' in bootstrap.parts:
        raise ValueError('bootstrap must use its explicit absolute prepared path')
    bootstrap_raw = prepare.read_file(bootstrap)
    require(manifest['bootstrap'], {'path': str(bootstrap), **identity(bootstrap_raw)}, 'bootstrap')
    commands = [{'profile': profile, 'cwd': str(workspace), 'executed': False,
                 'argv': [str(bootstrap), 'experimental', 'build-workspace', '--target',
                          prepare.TARGET, '--profile', profile, '--output', IMAGE]}
                for profile in prepare.PROFILES]
    require(manifest['buildCommands'], commands, 'prepared build commands')
    selected = [c for c in commands if c['argv'] == record['argv']]
    if len(selected) != 1:
        raise ValueError('build record argv/profile/output mismatch')
    profile = selected[0]['profile']
    require(record['cwd'], str(workspace), 'build record cwd')
    if type(record['exit']) is not int or record['exit'] != 0 or record['stderr'] != '':
        raise ValueError('build record requires exit 0 and empty stderr')
    if not isinstance(record['stdout'], str):
        raise ValueError('build record stdout must be text')
    if 'profile' in record:
        require(record['profile'], profile, 'build record profile')
    if 'output' in record:
        require(record['output'], IMAGE, 'build record output')
    payload = {'fairyc-state.exe': prepare.read_file(workspace / IMAGE)}
    require(record['image'], identity(payload['fairyc-state.exe']), 'build record image')
    windows_image(payload['fairyc-state.exe'])
    packaged = prepare.packs(root, inputs)
    require(manifest['packs'], [{'path': 'build/' + name, **identity(raw)}
                               for name, raw in packaged.items()], 'pack inventory')
    for name, raw in packaged.items():
        require(prepare.read_file(workspace / 'build' / name), raw, f'pack {name}')
        payload[name] = raw
    payload['LICENSE'] = prepare.read_file(root / 'LICENSE')
    payload['README.md'] = README
    # Do not copy raw records: their paths/stdout and optional fields are local.
    metadata = {
        'schema': 'fairyc.private-state-cli-zip/v1', 'scope': 'local-only private Windows developer snapshot',
        'target': prepare.TARGET, 'profile': profile,
        'snapshot': 'Exact current authored bytes; source hashes are not a commit claim',
        'inputs': manifest['inputs'], 'copiedFiles': manifest['copiedFiles'],
        'bootstrap': identity(bootstrap_raw),
        'build': {'argv': ['bootstrap.exe', *record['argv'][1:]], 'cwd': '.',
                  'pathPolicy': 'Bootstrap identity only; workspace cwd is relative, no local paths retained',
                  'exit': 0, 'stderr': '', 'image': record['image']},
        'files': [{'path': p, **identity(raw)} for p, raw in sorted(payload.items())],
        'acceptance': {'imageExecutedByPackager': False, 'cleanMachine': False,
                       'installed': False, 'linuxNative': False, 'hosted': False, 'publicRelease': False},
    }
    payload['metadata.json'] = json_bytes(metadata)
    payload['SHA256SUMS'] = ''.join(f'{prepare.digest(raw)}  {p}\n'
                                     for p, raw in sorted(payload.items())).encode('ascii')
    if (prepare.inventory(root) != inputs or prepare.read_file(bootstrap) != bootstrap_raw
            or prepare.read_file(workspace / 'prepare-manifest.json') != manifest_raw
            or prepare.read_file(record_path) != record_raw):
        raise ValueError('inputs changed during packaging')
    for source, destination in mappings:
        require(prepare.read_file(workspace / destination), inputs[source], f'changed copy {destination}')
    for name in ('fairyc-state.exe', *prepare.PACK_NAMES.values()):
        require(prepare.read_file(workspace / 'build' / name), payload[name], f'changed build {name}')
    return payload, metadata


def package(workspace, build_record, output, *, root=ROOT):
    root = prepare.no_links(root)
    workspace = cache_path(root, workspace)
    if '..' in Path(build_record).parts:
        raise ValueError('build record parent traversal is forbidden')
    build_record = prepare.no_links(build_record)
    output = cache_path(root, output)
    if output.suffix.lower() != '.zip' or output.is_relative_to(workspace):
        raise ValueError('output must be a new ZIP outside the prepared workspace')
    if output.exists():
        raise FileExistsError(f'output already exists: {output}')
    payload, metadata = validated_payload(root, workspace, build_record)
    output.parent.mkdir(parents=True, exist_ok=True)
    prepare.no_links(output)
    with output.open('xb') as stream:
        with zipfile.ZipFile(stream, 'w', compression=zipfile.ZIP_STORED) as archive:
            for name, raw in sorted(payload.items()):
                info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = 0o100644 << 16
                archive.writestr(info, raw)
    return metadata


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--workspace', type=Path, required=True)
    parser.add_argument('--build-record', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args(argv)
    try:
        package(args.workspace, args.build_record, args.output)
    except (OSError, ValueError, KeyError, TypeError, struct.error) as error:
        parser.error(str(error))
    print(args.output)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
