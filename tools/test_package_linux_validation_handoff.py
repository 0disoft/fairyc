"""Tiny Git fixtures only: passing Windows tests never proves native Linux."""

import importlib.util
import io
import json
import os
from pathlib import Path
import stat
import subprocess
import tarfile
import tempfile
import unittest
from unittest import mock
from types import SimpleNamespace
import zipfile

SPEC = importlib.util.spec_from_file_location(
    'linux_handoff', Path(__file__).with_name('package-linux-validation-handoff.py'))
handoff = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(handoff)
COLLECTOR_SOURCE = Path(__file__).with_name('collect-linux-native-validation.py').read_bytes()
EXPECTED = {
    'candidatebuild': ('internal/candidatebuild', (
        'TestLinuxPacketBuildAndPublication', 'TestLinuxPacketProcessArguments')),
    'trap-runtime': ('internal/cruntime', ('TestLinuxTrapReceiptProgressNative',)),
    'runtime': ('internal/cli', (
        'TestLinuxGNUCompanionRuntime', 'TestLinuxGNUCompanionRuntimeAssets',
        'TestLinuxGNUCompanionNative')),
    'fetch-host': ('internal/candidatehost', (
        'TestLinuxPackageDirectoryAdapterNative', 'TestLinuxPackageFileAdapterNative',
        'TestLinuxClockAdapterNative', 'TestLinuxHTTPSWorkerRejectsUntrustedNative',
        'TestLinuxHTTPSParentProcessNative', 'TestLinuxHTTPSParentRejectsUntrustedNative',
        'TestLinuxHTTPSParentTLSDeadlineNative')),
    'fetch-native': ('internal/cli', (
        'TestLinuxRemoteObservedReadNative', 'TestLinuxRemoteOfflineNative',
        'TestLinuxRemoteOnlineNative')),
    'cli': ('internal/cli', ('TestLinuxFileCLI',)),
}


class ArchiveCompressionTests(unittest.TestCase):
    def test_text_compression_preserves_bytes_modes_and_repeatability(self):
        source = b'compiler source repeated across authored fixtures\n' * 22000
        payload = {'source/fixture.txt': source, 'README.md': b'Unexecuted Linux plan\n'}
        entries = {'fixture.txt': {'mode': '100644'}}
        first = handoff.zip_bytes(payload, entries)
        self.assertEqual(first, handoff.zip_bytes(payload, entries))
        self.assertLess(len(first), len(source) // 10)
        with zipfile.ZipFile(io.BytesIO(first)) as archive:
            self.assertIsNone(archive.testzip())
            self.assertEqual(set(archive.namelist()), {handoff.PREFIX + path for path in payload})
            for name, body in payload.items():
                self.assertEqual(archive.read(handoff.PREFIX + name), body)
            self.assertEqual(archive.getinfo(handoff.PREFIX + 'source/fixture.txt').external_attr >> 16,
                             0o100644)


class HandoffTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name) / 'repo'
        self.root.mkdir()
        self.run_git('init', '-q')
        self.run_git('config', 'user.name', 'Fixture')
        self.run_git('config', 'user.email', 'fixture@example.invalid')
        self.run_git('config', 'core.autocrlf', 'false')
        self.write('.gitignore', b'.cache/\n')
        self.write('.gitattributes', b'* text=auto eol=lf\n*.dat -text\n')
        self.write('tracked.txt', b'committed LF\nsecond line\n')
        self.write('binary.dat', b'\0\x01\xff\r\n')
        self.write('scripts/executable.sh', b'#!/bin/sh\nprintf fixture\\n\n')
        self.write(handoff.COLLECTOR, COLLECTOR_SOURCE)
        for path in handoff.RECEIPTS:
            self.write(path, ('Authored receipt: ' + path + '\nNative unexecuted.\n').encode())
        owners = {}
        for package, roots in EXPECTED.values():
            owners.setdefault(package, []).extend(roots)
        for package, roots in owners.items():
            self.write(package + '/fixture_test.go', ('package fixture\n' + ''.join(
                'func ' + root + '(t *testing.T) {}\n' for root in roots)).encode())
        self.commit()
        self.run_git('update-index', '--chmod=+x', 'scripts/executable.sh')
        self.run_git('commit', '-qm', 'track executable mode')
        # Windows cannot represent Git's executable bit; Git's index retains it.
        self.run_git('config', 'core.filemode', 'false')

    def run_git(self, *args, data=None):
        return subprocess.run(['git', *args], cwd=self.root, input=data, check=True,
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout

    def write(self, name, data):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)

    def commit(self):
        self.run_git('add', '--all')
        self.run_git('commit', '-qm', 'fixture snapshot')

    def build(self, name='bundle'):
        result = handoff.build(self.root, Path('.cache') / name, timeout=321)
        path = self.root / '.cache' / name / 'handoff.zip'
        with zipfile.ZipFile(path) as archive:
            payload = {name.removeprefix(handoff.PREFIX): archive.read(name)
                       for name in archive.namelist()}
        return result, path.read_bytes(), payload

    def test_blob_integrity_manifest_receipts_and_determinism(self):
        # Ignored local files never enter the archive.
        self.write('.cache/private/token', b'do not include')
        self.run_git('config', 'core.autocrlf', 'true')
        self.write('tracked.txt', b'committed LF\r\nsecond line\r\n')
        # Refresh only the tiny fixture index after changing checkout EOL policy;
        # its blob/revision remain unchanged, while worktree bytes stay CRLF.
        self.run_git('add', '--', 'tracked.txt')
        self.assertFalse(self.run_git('status', '--porcelain'))
        result, first, payload = self.build('first')
        second_result, second, _ = self.build('second')
        self.assertEqual(first, second)
        self.assertEqual(result, second_result)
        self.assertEqual(result['sha256'], handoff.digest(first))
        self.assertEqual(result['bytes'], len(first))
        manifest = json.loads(payload['manifest.json'])
        expected_names = self.run_git('ls-files', '-z').decode().strip('\0').split('\0')
        self.assertEqual(manifest['sourceFileCount'], len(expected_names))
        self.assertEqual(result['sourceFileCount'], len(expected_names))
        self.assertEqual(result['zipEntryCount'], len(payload))
        self.assertFalse(manifest['linuxNativeExecuted'])
        self.assertEqual(manifest['sourceRevision'], self.run_git('rev-parse', 'HEAD').decode().strip())
        for name in expected_names:
            self.assertEqual(payload['source/' + name], self.run_git('show', 'HEAD:' + name))
        self.assertEqual(payload['source/tracked.txt'], b'committed LF\nsecond line\n')
        self.assertEqual(payload['source/binary.dat'], b'\0\x01\xff\r\n')
        executable = next(item for item in manifest['files']
                          if item['path'] == 'source/scripts/executable.sh')
        self.assertEqual(executable['mode'], '100755')
        self.assertEqual(manifest['sourceBytes'], sum(len(payload['source/' + name]) for name in expected_names))
        for item in manifest['files']:
            data = payload[item['path']]
            self.assertEqual(handoff.digest(data), item['sha256'])
            self.assertEqual(len(data), item['bytes'])
            if item['path'].startswith('source/'):
                self.assertEqual(handoff.object_hash('blob', data, 'sha1'), item['gitBlob'])
        sums = {}
        for line in payload['SHA256SUMS'].decode().splitlines():
            checksum, name = line.split('  ', 1)
            sums[name] = checksum
            self.assertEqual(handoff.digest(payload[name]), checksum)
        self.assertEqual(set(sums), set(payload) - {'SHA256SUMS'})
        self.assertFalse(any('.git/' in name or '.cache/' in name for name in payload))
        for path in handoff.RECEIPTS:
            self.assertIn('source/' + path, payload)
        with zipfile.ZipFile(io.BytesIO(first)) as archive:
            self.assertEqual(archive.namelist(), sorted(archive.namelist()))
            for info in archive.infolist():
                self.assertEqual(info.date_time, (1980, 1, 1, 0, 0, 0))
            self.assertEqual(archive.getinfo(handoff.PREFIX + 'source/scripts/executable.sh').external_attr >> 16,
                             0o100755)

    def test_exact_grouped_argv_relative_plan_and_pending_observations(self):
        # The fixture collector throws if its execution API is accidentally used.
        self.write(handoff.COLLECTOR, COLLECTOR_SOURCE +
                   b'\ndef collect(*args, **kwargs):\n    raise AssertionError("must never execute")\n')
        self.commit()
        _, _, payload = self.build()
        plan = json.loads(payload['plan.json'])
        self.assertEqual(plan['cwd'], '.')
        self.assertEqual(plan['mode'], 'plan')
        self.assertEqual(plan['groups'], list(EXPECTED))
        self.assertEqual(len(plan['commands']), len(EXPECTED))
        for command, (group, (package, roots)) in zip(plan['commands'], EXPECTED.items()):
            self.assertEqual(command['group'], group)
            self.assertEqual(command['roots'], list(roots))
            self.assertEqual(command['argv'], ['go', 'test', '-mod=readonly', '-count=1', '-json',
                             '-timeout', '321s', './' + package, '-run', '^(' + '|'.join(roots) + ')$'])
            self.assertEqual(command['timeoutSeconds'], 351)
            self.assertEqual(command['sources'], {name: package + '/fixture_test.go' for name in roots})
        remaining = json.loads(payload['remaining-observations.json'])
        self.assertFalse(remaining['complete'])
        self.assertFalse(remaining['linuxNativeExecuted'])
        self.assertEqual({item['id'] for item in remaining['remaining']}, {
            'trusted-tls', 'atlas-stdio-pipes', 'cold-warm-cache',
            'consecutive-self-build-full-acceptance'})
        for name in ('manifest.json', 'plan.json', 'remaining-observations.json', 'README.md'):
            text = payload[name].decode()
            self.assertNotIn(str(self.root), text)
            self.assertNotIn(self.root.as_posix(), text)
        self.assertIn(b'--execute', payload['README.md'])
        self.assertIn(b'UNEXECUTED', payload['README.md'])

    def test_dirty_tracked_staged_and_untracked_are_refused(self):
        for kind in ('tracked', 'staged', 'untracked'):
            with self.subTest(kind=kind):
                if kind == 'untracked':
                    self.write('new.txt', b'new\n')
                else:
                    self.write('tracked.txt', b'changed\n')
                    if kind == 'staged':
                        self.run_git('add', 'tracked.txt')
                with self.assertRaisesRegex(ValueError, 'dirty'):
                    self.build(kind)
                self.assertFalse((self.root / '.cache' / kind).exists())
                self.run_git('reset', '--hard', '-q', 'HEAD')
                (self.root / 'new.txt').unlink(missing_ok=True)

    def test_hidden_index_changes_are_refused(self):
        for flag in ('assume-unchanged', 'skip-worktree'):
            with self.subTest(flag=flag):
                self.run_git('update-index', '--' + flag, 'tracked.txt')
                with self.assertRaisesRegex(ValueError, 'index entries'):
                    self.build(flag)
                self.run_git('update-index', '--no-' + flag, 'tracked.txt')

    def test_unsafe_or_existing_output_is_refused(self):
        for value in ('.cache', 'outside', '../escape', '.cache/../escape',
                      self.root.parent / 'outside', '.cache/name:stream', '.cache/NUL.zip'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                handoff.build(self.root, value)
        self.build('exists')
        before = (self.root / '.cache/exists/handoff.zip').read_bytes()
        with self.assertRaisesRegex(ValueError, 'already exists'):
            self.build('exists')
        self.assertEqual(before, (self.root / '.cache/exists/handoff.zip').read_bytes())
        self.write('.cache/file', b'not a directory')
        with self.assertRaisesRegex(ValueError, 'non-directory'):
            handoff.build(self.root, '.cache/file/child')

    def test_output_reparse_refusal_with_mocked_windows_attribute(self):
        (self.root / '.cache').mkdir()
        original = Path.lstat

        def lstat(path, *args, **kwargs):
            if path == self.root / '.cache':
                return SimpleNamespace(st_mode=stat.S_IFDIR, st_file_attributes=0x400)
            return original(path, *args, **kwargs)

        with mock.patch.object(Path, 'lstat', lstat):
            with self.assertRaisesRegex(ValueError, 'reparse'):
                self.build()

    def test_exclusive_creation_does_not_remove_a_foreign_leaf(self):
        original = handoff.portable_plan

        def plan(sources, output, timeout):
            result = original(sources, output, timeout)
            (output / 'handoff.zip').write_bytes(b'foreign leaf')
            return result

        with mock.patch.object(handoff, 'portable_plan', side_effect=plan):
            with self.assertRaises(FileExistsError):
                self.build()
        self.assertEqual((self.root / '.cache/bundle/handoff.zip').read_bytes(), b'foreign leaf')

    def test_source_symlink_submodule_and_unsafe_git_paths_are_refused(self):
        for mode, kind, path in (('120000', 'blob', 'link'), ('160000', 'commit', 'submodule'),
                                 ('100644', 'blob', '../outside'), ('100644', 'blob', 'AUX.txt'),
                                 ('100644', 'blob', 'C:/outside'), ('100644', 'blob', 'bad\\path'),
                                 ('100644', 'blob', 'a/./b'), ('100644', 'blob', '/absolute')):
            with self.subTest(path=path), self.assertRaises(ValueError):
                handoff.tree_entries(f'{mode} {kind} {"a" * 40}\t{path}\0'.encode())
        raw = b'100644 blob ' + b'a' * 40
        with self.assertRaisesRegex(ValueError, 'case-aliased'):
            handoff.tree_entries(raw + b'\tfile\0' + raw + b'\tFILE\0')

    def test_dangerous_tar_entries_and_byte_substitution_are_refused(self):
        data = b'original'
        entries = {'safe': {'mode': '100644', 'gitBlob': handoff.object_hash('blob', data, 'sha1')}}
        for name, kind, body in (('../escape', tarfile.REGTYPE, data),
                                  ('/absolute', tarfile.REGTYPE, data),
                                  ('safe', tarfile.SYMTYPE, b''), ('safe', tarfile.LNKTYPE, b''),
                                  ('safe', tarfile.REGTYPE, b'substituted')):
            with self.subTest(name=name, kind=kind):
                stream = io.BytesIO()
                with tarfile.open(fileobj=stream, mode='w') as archive:
                    item = tarfile.TarInfo(name)
                    item.type, item.size, item.linkname = kind, len(body), '../escape'
                    archive.addfile(item, io.BytesIO(body))
                with self.assertRaises(ValueError):
                    handoff.archive_sources(stream.getvalue(), entries, 'sha1')

    def test_export_ignore_and_export_subst_refused(self):
        for attribute in ('tracked.txt export-ignore', 'tracked.txt export-subst'):
            with self.subTest(attribute=attribute):
                self.write('.gitattributes', (attribute + '\n').encode())
                self.write('tracked.txt', b'$Format:%H$\n')
                self.commit()
                with self.assertRaisesRegex(ValueError, 'omitted|differ'):
                    self.build()

    def test_committed_artifacts_and_credentials_are_refused(self):
        for name, body in (('.cache/forced.txt', b'cache'), ('offlinearchives/data.zip', b'archive'),
                           ('_toolchain/compiler', b'tool'), ('.env', b'private'),
                           ('private.txt', b'-----BEGIN PRIVATE KEY-----\nprivate\n')):
            with self.subTest(name=name):
                self.write(name, body)
                self.run_git('add', '-f', '--', name)
                self.run_git('commit', '-qm', 'forbidden fixture')
                with self.assertRaisesRegex(ValueError, 'refusing'):
                    self.build()
                self.run_git('rm', '-f', '--', name)
                self.run_git('commit', '-qm', 'remove forbidden fixture')

    def test_missing_roots_refused_and_no_partial_bundle_remains(self):
        self.write('internal/cruntime/fixture_test.go', b'package fixture\n')
        self.commit()
        with self.assertRaisesRegex(ValueError, 'missing or ambiguous'):
            self.build()
        self.assertFalse((self.root / '.cache/bundle').exists())

    def test_revision_moving_during_packaging_refused(self):
        revision = self.run_git('rev-parse', 'HEAD').decode().strip()
        with mock.patch.object(handoff, 'clean_revision', side_effect=[revision, 'f' * 40]):
            with self.assertRaisesRegex(ValueError, 'revision changed'):
                self.build()
        self.assertFalse((self.root / '.cache/bundle').exists())

    def test_readme_commands_restore_original_commit_and_clean_tree(self):
        _, _, payload = self.build()
        receiver = Path(self.temporary.name) / 'receiver'
        receiver.mkdir()
        for name, body in payload.items():
            path = receiver / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(body)
        script = payload['README.md'].decode().split("python3 -B - <<'PY'\n", 1)[1].split('\nPY\n', 1)[0]
        # Run the exact README code with a recorder around the real chmod.
        recording = '''import os, json
chmod_calls = []
real_chmod = os.chmod
def recorded_chmod(path, mode):
    chmod_calls.append([str(path), mode])
    return real_chmod(path, mode)
os.chmod = recorded_chmod
'''
        completed = subprocess.run([os.sys.executable, '-B', '-c', recording + script +
                                   '\nprint(json.dumps(chmod_calls))'], cwd=receiver, check=True,
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        calls = json.loads(completed.stdout)
        manifest = json.loads(payload['manifest.json'])
        expected = {str(receiver / f['path']): int(f['mode'], 8) & 0o777
                    for f in manifest['files'] if f['path'].startswith('source/')}
        self.assertEqual(dict(calls), expected)
        self.assertEqual(len(calls), len(expected))
        actual = subprocess.run(['git', 'rev-parse', 'HEAD'], cwd=receiver / 'source', check=True,
                                stdout=subprocess.PIPE).stdout
        self.assertEqual(actual, self.run_git('rev-parse', 'HEAD'))
        self.assertFalse(subprocess.run(['git', 'status', '--porcelain'], cwd=receiver / 'source',
                                       stdout=subprocess.PIPE, check=True).stdout)
        mode = subprocess.run(['git', 'ls-files', '--stage', 'scripts/executable.sh'],
                              cwd=receiver / 'source', stdout=subprocess.PIPE, check=True).stdout
        self.assertTrue(mode.startswith(b'100755 '))

    def test_readme_refuses_links_and_unsafe_paths_before_any_chmod(self):
        _, _, payload = self.build()
        script = payload['README.md'].decode().split("python3 -B - <<'PY'\n", 1)[1].split('\nPY\n', 1)[0]
        for case in ('hardlink', 'traversal', 'reparse'):
            receiver = Path(self.temporary.name) / case
            receiver.mkdir()
            for name, body in payload.items():
                path = receiver / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(body)
            manifest_path = receiver / 'manifest.json'
            if case == 'hardlink':
                os.link(receiver / 'source/tracked.txt', receiver / 'foreign-hardlink')
            elif case == 'traversal':
                manifest = json.loads(manifest_path.read_text())
                manifest['files'][0]['path'] = '../outside'
                manifest_path.write_text(json.dumps(manifest))
            guard = 'import os\nos.chmod = lambda *args: (_ for _ in ()).throw(AssertionError("chmod called"))\n'
            if case == 'reparse':
                guard += '''import pathlib, types
original_lstat = pathlib.Path.lstat
def reparse_lstat(path, *args, **kwargs):
    value = original_lstat(path, *args, **kwargs)
    if path.name == 'tracked.txt':
        return types.SimpleNamespace(st_mode=value.st_mode, st_file_attributes=0x400)
    return value
pathlib.Path.lstat = reparse_lstat
'''
            result = subprocess.run([os.sys.executable, '-B', '-c', guard + script], cwd=receiver,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertNotEqual(result.returncode, 0)
            self.assertNotIn(b'chmod called', result.stderr)
            self.assertFalse((receiver / 'source/.git').exists())


if __name__ == '__main__':
    unittest.main()
