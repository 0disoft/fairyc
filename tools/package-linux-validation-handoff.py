"""Package a clean committed source snapshot and an UNEXECUTED Linux plan."""

import argparse
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import shlex
import stat
import subprocess
import sys
import tarfile
import tempfile
import unicodedata
import zipfile

ROOT = Path(__file__).resolve().parents[1]
PREFIX = 'fairyc-linux-validation/'
COLLECTOR = 'tools/collect-linux-native-validation.py'
GROUPS = ('candidatebuild', 'trap-runtime', 'runtime', 'fetch-host', 'fetch-native', 'cli')
RECEIPTS = (
    'docs/development/linux-native-batch-20261107.md',
    'docs/development/candidate-linux-package-20261107.md',
    'docs/development/candidate-linux-runtime-sync-20261106.md',
    'docs/development/candidate-linux-package-fetch-20261106.md',
    'docs/development/candidate-linux-fetch-cli-20261106.md',
    'docs/development/candidate-linux-atlas-cli-20261106.md',
    'docs/development/candidate-linux-atlas-edit-20261106.md',
    'docs/abi/linux-toolchain-v0.md',
)
FORBIDDEN_DIRS = {
    '.git', '.cache', 'build', 'node_modules', 'dist', '__pycache__',
    '.venv', 'venv', '_toolchain', 'toolchains', 'offlinearchives',
    'offline-archives', 'offline_archives', 'secrets', '.ssh', '.aws',
}
FORBIDDEN_SUFFIXES = {
    '.zip', '.tar', '.gz', '.tgz', '.xz', '.7z', '.rar', '.exe', '.dll',
    '.so', '.o', '.obj', '.pyc', '.pfx', '.p12', '.pem', '.key',
}
SECRET = re.compile(
    rb'(?m)^-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----|'
    rb'\bAKIA[0-9A-Z]{16}\b|\bsk-proj-[A-Za-z0-9_-]{30,}')


def json_bytes(value):
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + '\n').encode()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def git(root, *args):
    result = subprocess.run(
        ['git', '--no-optional-locks', '-c', 'core.fsmonitor=false',
         '-c', 'core.untrackedCache=false', *args], cwd=root,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
    if result.returncode:
        # Do not echo subprocess stderr, configuration or credential-bearing URLs.
        raise ValueError('Git operation failed: ' + args[0])
    return result.stdout


def clean_revision(root):
    if git(root, 'status', '--porcelain=v1', '-z', '--untracked-files=all',
           '--ignore-submodules=none'):
        raise ValueError('refusing dirty tracked or nonignored untracked source tree')
    for record in git(root, 'ls-files', '-v', '-z').split(b'\0'):
        if record and (record[:1].islower() or record.startswith(b'S ')):
            raise ValueError('refusing assume-unchanged or skip-worktree index entries')
    return git(root, 'rev-parse', '--verify', 'HEAD^{commit}').decode().strip()


def safe_path(name):
    """Use one portable spelling for Git, tar, staging and ZIP paths."""
    if (not name or name.startswith('/') or unicodedata.normalize('NFC', name) != name
            or re.search(r'[\\<>:"|?*\x00-\x1f\x7f]', name)):
        raise ValueError('unsafe archive path')
    for part in name.split('/'):
        if (part in ('', '.', '..') or part.endswith(('.', ' ')) or
                re.fullmatch(r'(?i:con|prn|aux|nul|com[1-9]|lpt[1-9])(?:\..*)?', part)):
            raise ValueError('unsafe archive path')
    return PurePosixPath(name)


def source_path(name):
    path = safe_path(name)
    if (any(part.casefold() in FORBIDDEN_DIRS for part in path.parts) or
            path.suffix.casefold() in FORBIDDEN_SUFFIXES or
            path.name.casefold().startswith('.env') or
            path.name.casefold() in ('credentials', 'credentials.json', 'id_rsa', 'id_ed25519')):
        raise ValueError('refusing tracked cache, archive, toolchain or credential path: ' + name)
    return path


def object_hash(kind, data, algorithm):
    return hashlib.new(algorithm, kind.encode() + b' ' + str(len(data)).encode() + b'\0' + data).hexdigest()


def tree_entries(raw):
    entries, spellings = {}, set()
    for record in raw.split(b'\0'):
        if not record:
            continue
        metadata, path = record.split(b'\t', 1)
        mode, kind, oid = metadata.decode('ascii').split()
        name = path.decode('utf-8', errors='strict')
        source_path(name)
        if mode not in ('100644', '100755') or kind != 'blob':
            raise ValueError('symlinks, submodules and nonregular Git entries are refused')
        if name.casefold() in spellings:
            raise ValueError('duplicate or case-aliased Git path')
        spellings.add(name.casefold())
        entries[name] = {'mode': mode, 'gitBlob': oid}
    return entries


def archive_sources(raw, entries, algorithm):
    """No extraction: archive attributes must not omit or substitute any blob."""
    sources, seen = {}, set()
    with tarfile.open(fileobj=io.BytesIO(raw), mode='r:') as archive:
        for member in archive:
            name = member.name[:-1] if member.isdir() and member.name.endswith('/') else member.name
            safe_path(name)
            if name in seen:
                raise ValueError('duplicate tar path')
            seen.add(name)
            if member.isdir():
                if not any(path.startswith(name + '/') for path in entries):
                    raise ValueError('unexpected tar directory')
                continue
            if not member.isfile() or name not in entries:
                raise ValueError('unexpected or nonregular tar entry')
            data = archive.extractfile(member).read()
            if object_hash('blob', data, algorithm) != entries[name]['gitBlob']:
                raise ValueError('archive bytes differ from committed Git blob: ' + name)
            if SECRET.search(data):
                raise ValueError('refusing possible tracked credential material: ' + name)
            sources[name] = data
    if set(sources) != set(entries):
        raise ValueError('archive omitted tracked source (check export-ignore attributes)')
    return sources


def is_link(path):
    info = path.lstat()
    return stat.S_ISLNK(info.st_mode) or bool(
        getattr(info, 'st_file_attributes', 0) & getattr(stat, 'FILE_ATTRIBUTE_REPARSE_POINT', 0x400))


def safe_output(root, output):
    if '..' in Path(output).parts:
        raise ValueError('output traversal is refused')
    target = Path(output) if Path(output).is_absolute() else root / output
    target = Path(os.path.abspath(target))
    try:
        parts = target.relative_to(root).parts
    except ValueError:
        raise ValueError('output must be a new named directory under repository .cache') from None
    if len(parts) < 2 or parts[0] != '.cache':
        raise ValueError('output must be a new named directory under repository .cache')
    safe_path('/'.join(parts))
    cursor = root
    for part in parts:
        cursor = cursor / part
        if os.path.lexists(cursor) and (is_link(cursor) or not cursor.is_dir()):
            raise ValueError('output contains a link, reparse point or non-directory')
    if os.path.lexists(target):
        raise ValueError('output already exists; never overwrite handoffs')
    # Without this ignore rule creating output would itself invalidate cleanliness.
    probe = subprocess.run(['git', 'check-ignore', '-q', '--no-index', '--',
                            target.relative_to(root).as_posix() + '/probe'], cwd=root)
    if probe.returncode != 0:
        raise ValueError('output must be Git-ignored under .cache')
    return target


def check_output_identity(root, target, identity):
    cursor = root
    for part in target.relative_to(root).parts:
        cursor = cursor / part
        if is_link(cursor) or not cursor.is_dir():
            raise ValueError('output changed to a link or reparse point')
    info = target.stat()
    if (info.st_dev, info.st_ino) != identity:
        raise ValueError('output directory identity changed')


def create_output(root, target):
    cursor = root
    for part in target.relative_to(root).parts:
        cursor = cursor / part
        if not os.path.lexists(cursor):
            cursor.mkdir(exist_ok=False)
        elif cursor == target:
            raise ValueError('output already exists; never overwrite handoffs')
        if is_link(cursor) or not cursor.is_dir():
            raise ValueError('output contains a link or reparse point')


def portable_plan(sources, output, timeout):
    with tempfile.TemporaryDirectory(prefix='plan-', dir=output) as temporary:
        root = Path(temporary)
        for name, data in sources.items():
            if name == COLLECTOR or (name.startswith('internal/') and name.endswith('_test.go')):
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
        spec = importlib.util.spec_from_file_location('handoff_collector', root / COLLECTOR)
        collector = importlib.util.module_from_spec(spec)
        # Avoid __pycache__ output from an imported committed planner.
        code = compile(sources[COLLECTOR], COLLECTOR, 'exec')
        exec(code, collector.__dict__)
        plan = collector.make_plan(root, list(GROUPS), timeout)
        plan['cwd'] = '.'
    if plan['mode'] != 'plan' or plan['groups'] != list(GROUPS):
        raise ValueError('unexpected collector plan shape')
    for command in plan['commands']:
        for source in command['sources'].values():
            safe_path(source)
            if source not in sources:
                raise ValueError('plan owner missing from source snapshot')
    return plan


def remaining_observations():
    return {
        'linuxNativeExecuted': False, 'complete': False,
        'scope': 'All selected roots are authored, unexecuted Linux replay; no native pass claimed.',
        'remaining': [
            {'id': 'trusted-tls', 'status': 'unexecuted; not established by selected roots',
             'observation': 'Positive trusted HTTPS path with real libcurl, CA trust, headers/body and publication.'},
            {'id': 'atlas-stdio-pipes', 'status': 'unexecuted; authored within cli group',
             'observation': 'Atlas CLI/MCP stdio, response before EOF, default SIGPIPE and closed-pipe exit/error behavior.'},
            {'id': 'cold-warm-cache', 'status': 'unexecuted; bounded cases within cli/fetch groups',
             'observation': 'Real Linux cold publication, warm reuse, corruption and offline/no-network behavior.'},
            {'id': 'consecutive-self-build-full-acceptance', 'status': 'unexecuted; outside collector completion',
             'observation': 'Consecutive full Linux self-builds, broad Go/Fairy parity and full distribution acceptance.'},
        ],
        'authoredReceipts': ['source/' + path for path in RECEIPTS],
    }


def readme(revision, tree, algorithm, timeout):
    options = ' '.join('--group ' + shlex.quote(group) for group in GROUPS)
    return f'''# Fairy C Linux source and validation handoff

Source revision: `{revision}`; Git tree: `{tree}` ({algorithm}).
UNEXECUTED: no Linux run, TLS success, self-build or native pass is recorded here.
`source/` contains every tracked regular blob, byte-for-byte from this commit.
Git archive output was checked against every tree blob, including LF bytes on
Windows. No Git directory, cache, toolchain or offline archive is included.
`plan.json` is the existing collector's plan, with cwd `.` relative to `source/`.
Read `remaining-observations.json` and the exact authored receipts it lists.
Those historical receipts do not become same-commit validation passes.

## Verify and prepare on the Linux receiver

Use a NEW extraction directory; do not merge with another checkout. Only extract
this verified fixed ZIP: the builder rejects links and unsafe/case-aliased paths.
Check the external handoff.zip.sha256 before extraction. Linux `unzip` preserves
the ZIP's Unix executable mode. Python ZipFile extraction does NOT restore modes;
the restoration code below explicitly applies each verified manifest Git mode
(100644 -> 0644, 100755 -> 0755) to source files inside this extraction only.
From this directory (beside this README), verify every payload before changing source:

```sh
sha256sum -c SHA256SUMS
```

The source archive intentionally has no `.git`. The collector probes Git HEAD
and cleanliness. Restore minimal Git context at the exact original commit with
the following commands, only in a fresh extraction. This imports the original
commit object, not a synthetic validation commit. Ancestor history is absent.

```sh
python3 -B - <<'PY'
import hashlib, json, os, pathlib, stat, subprocess
base = pathlib.Path.cwd().resolve()
m = json.loads((base / 'manifest.json').read_text())
def git(*args, data=None):
    return subprocess.run(['git', '-c', 'core.autocrlf=false', *args],
                          cwd='source', input=data, check=True,
                          stdout=subprocess.PIPE).stdout.decode().strip()
if (base / 'source/.git').exists():
    raise SystemExit('refusing existing Git context')
# Validate every component and byte BEFORE chmod. No links, hard links, reparse
# points, absolute names or traversal; only regular files in this extraction.
seen = set()
for f in m['files']:
    relative = pathlib.PurePosixPath(f['path'])
    if (relative.is_absolute() or relative.as_posix() != f['path'] or
            any(p in ('', '.', '..') for p in relative.parts) or
            '\\\\' in f['path'] or ':' in f['path'] or f['path'] in seen):
        raise SystemExit('unsafe manifest path')
    seen.add(f['path'])
    path = base
    for part in relative.parts:
        path = path / part
        info = path.lstat()
        if stat.S_ISLNK(info.st_mode) or getattr(info, 'st_file_attributes', 0) & 0x400:
            raise SystemExit('link or reparse point refused')
    if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
        raise SystemExit('nonregular or hard-linked payload refused')
    body = path.read_bytes()
    if len(body) != f['bytes'] or hashlib.sha256(body).hexdigest() != f['sha256']:
        raise SystemExit('payload checksum mismatch')
    if f['path'].startswith('source/') and f.get('mode') not in ('100644', '100755'):
        raise SystemExit('unsupported Git mode')
for f in m['files']:
    if f['path'].startswith('source/'):
        os.chmod(base / f['path'], int(f['mode'], 8) & 0o777)
git('init', '--object-format=' + m['gitObjectFormat'])
git('config', 'core.autocrlf', 'false')
git('add', '--force', '--all')
index = b''.join((f"{{f['mode']}} {{f['gitBlob']}}\\t{{f['path'][7:]}}\\0").encode()
                 for f in m['files'] if f['path'].startswith('source/'))
git('update-index', '-z', '--index-info', data=index)
if git('write-tree') != m['sourceTree']:
    raise SystemExit('source tree mismatch')
commit = git('hash-object', '-t', 'commit', '-w', '--stdin',
             data=(base / 'snapshot.commit').read_bytes())
if commit != m['sourceRevision']:
    raise SystemExit('commit mismatch')
git('update-ref', '--no-deref', 'HEAD', commit)
git('config', 'core.filemode', 'false')
if git('status', '--porcelain', '--untracked-files=all'):
    raise SystemExit('dirty extracted source')
PY
cd source
```

Prerequisites must be prepared separately: native Ubuntu 24.04 amd64 reference,
Go version from `go.mod`, Python 3, Git, GCC >=13, Clang >=18, development/linker/
sanitizer libraries, pkg-config, libcurl development/runtime and trusted CA store.
Use `docs/abi/linux-toolchain-v0.md`. Dependencies and Go toolchain must already
be locally available; the collector forces GOTOOLCHAIN=local, GOPROXY=off and
GOSUMDB=off. No bundled installer, dependency cache or toolchain is supplied.

## Plan and later authorized native execution

The first command only plans. The second is for the separately authorized Linux
batch; neither command was run by this packager. Each output must be new.

```sh
python3 -B tools/collect-linux-native-validation.py {options} --timeout-seconds {timeout} --output .cache/linux-handoff-plan
python3 -B tools/collect-linux-native-validation.py --execute {options} --timeout-seconds {timeout} --output .cache/linux-handoff-native
```

The six groups use exact anchored existing roots; no implicit ./... matrix.
`cli` includes the expensive full Linux CLI (GCC low/optimized), Atlas and MCP
closed-pipe replay. Separate Windows Linux-help command probes are not Linux
roots. Selected fetch tests do not establish a positive trusted TLS download.
Check raw logs, skips, signals, tool probes and receipt completeness. Collector
`complete` is bounded to selected roots, not final acceptance. Trusted TLS,
default-SIGPIPE observations, cold/warm caches and consecutive self-build/full
acceptance remain explicit pending observations until real evidence is reviewed.
No push, dispatch, release, provisioning or Linux execution occurs during packaging.
'''.encode()


def zip_bytes(payload, entries):
    result = io.BytesIO()
    with zipfile.ZipFile(result, 'w', compression=zipfile.ZIP_STORED) as archive:
        for name in sorted(payload):
            safe_path(name)
            info = zipfile.ZipInfo(PREFIX + name, date_time=(1980, 1, 1, 0, 0, 0))
            info.create_system = 3
            mode = int(entries[name[7:]]['mode'], 8) if name.startswith('source/') else 0o100644
            info.external_attr = mode << 16
            archive.writestr(info, payload[name])
    return result.getvalue()


def build(root, output, timeout=3600):
    root = Path(root).resolve(strict=True)
    if not 1 <= timeout <= 3600:
        raise ValueError('timeout must be 1..3600 seconds')
    if Path(git(root, 'rev-parse', '--show-toplevel').decode().strip()).resolve() != root:
        raise ValueError('root must be the Git repository root')
    revision = clean_revision(root)
    target = safe_output(root, output)
    algorithm = git(root, 'rev-parse', '--show-object-format').decode().strip()
    if algorithm not in ('sha1', 'sha256'):
        raise ValueError('unsupported Git object format')
    tree = git(root, 'rev-parse', revision + '^{tree}').decode().strip()
    entries = tree_entries(git(root, 'ls-tree', '-r', '-z', '--full-tree', revision))
    sources = archive_sources(git(root, 'archive', '--format=tar', revision), entries, algorithm)
    if any(path not in sources for path in (COLLECTOR, *RECEIPTS)):
        raise ValueError('required committed collector or authored Linux receipt missing')
    commit = git(root, 'cat-file', 'commit', revision)
    if object_hash('commit', commit, algorithm) != revision:
        raise ValueError('commit object mismatch')
    create_output(root, target)
    info = target.stat()
    identity = (info.st_dev, info.st_ino)
    created = []
    try:
        plan = portable_plan(sources, target, timeout)
        payload = {'source/' + name: data for name, data in sources.items()}
        payload.update({'plan.json': json_bytes(plan), 'snapshot.commit': commit,
                        'remaining-observations.json': json_bytes(remaining_observations()),
                        'README.md': readme(revision, tree, algorithm, timeout)})
        files = []
        for name in sorted(payload):
            item = {'path': name, 'sha256': digest(payload[name]), 'bytes': len(payload[name])}
            if name.startswith('source/'):
                item.update(entries[name[7:]])
            files.append(item)
        manifest = {'schemaVersion': 'fairyc.linux-validation-handoff/v1',
                    'sourceRevision': revision, 'sourceTree': tree, 'gitObjectFormat': algorithm,
                    'sourceFileCount': len(sources), 'sourceBytes': sum(map(len, sources.values())),
                    'linuxNativeExecuted': False, 'planCwd': '.', 'files': files}
        payload['manifest.json'] = json_bytes(manifest)
        payload['SHA256SUMS'] = ''.join(digest(payload[name]) + '  ' + name + '\n'
                                       for name in sorted(payload)).encode()
        data = zip_bytes(payload, entries)
        if clean_revision(root) != revision:
            raise ValueError('source revision changed during packaging')
        check_output_identity(root, target, identity)
        # Exclusive creation also refuses an existing file or symlink at the leaf.
        with (target / 'handoff.zip').open('xb') as stream:
            created.append(('handoff.zip', os.fstat(stream.fileno())))
            stream.write(data)
        check_output_identity(root, target, identity)
        with (target / 'handoff.zip.sha256').open('x', encoding='ascii', newline='\n') as stream:
            created.append(('handoff.zip.sha256', os.fstat(stream.fileno())))
            stream.write(digest(data) + '  handoff.zip\n')
        if clean_revision(root) != revision:
            raise ValueError('source revision changed during packaging')
        return {'sourceRevision': revision, 'sourceTree': tree, 'sourceFileCount': len(sources),
                'zipEntryCount': len(payload), 'sha256': digest(data), 'bytes': len(data),
                'linuxNativeExecuted': False}
    except BaseException:
        # Never remove foreign leaves or follow a replaced/reparse output directory.
        check_output_identity(root, target, identity)
        for name, owned in created:
            path = target / name
            current = path.lstat()
            if (current.st_dev, current.st_ino) == (owned.st_dev, owned.st_ino):
                path.unlink()
        if not any(target.iterdir()):
            target.rmdir()
        raise


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True,
                        help='new Git-ignored directory under repository .cache')
    parser.add_argument('--timeout-seconds', type=int, default=3600)
    args = parser.parse_args(argv)
    try:
        result = build(ROOT, args.output, args.timeout_seconds)
    except (OSError, ValueError, tarfile.TarError, UnicodeError) as error:
        parser.error(str(error))
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == '__main__':
    sys.exit(main())
