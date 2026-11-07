"""Local temporary-checkout packaging tests; no native build or image execution."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import stat
import struct
import types
import unittest
from unittest import mock
import zipfile

TOOLS = Path(__file__).resolve().parent


def load(name, leaf):
    spec = importlib.util.spec_from_file_location(name, TOOLS / leaf)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


package = load('package_state_cli', 'package-candidate-state-cli.py')
fixtures = load('prepare_state_cli_tests', 'test_prepare_candidate_state_cli.py')
prepare = package.prepare


def pe_image():
    """Small structural PE32+ fixture, never executed or called native evidence."""
    raw = bytearray(1024)
    raw[:2] = b'MZ'
    struct.pack_into('<I', raw, 0x3c, 0x80)
    raw[0x80:0x84] = b'PE\0\0'
    struct.pack_into('<HHIIIHH', raw, 0x84, 0x8664, 1, 0, 0, 0, 240, 0x22)
    optional = 0x98
    struct.pack_into('<H', raw, optional, 0x20b)
    struct.pack_into('<I', raw, optional + 16, 0x1000)
    struct.pack_into('<Q', raw, optional + 24, 0x140000000)
    struct.pack_into('<II', raw, optional + 32, 0x1000, 0x200)
    struct.pack_into('<II', raw, optional + 56, 0x2000, 512)
    struct.pack_into('<H', raw, optional + 68, 3)
    struct.pack_into('<I', raw, optional + 108, 16)
    section = optional + 240
    raw[section:section + 8] = b'.text\0\0\0'
    struct.pack_into('<IIII', raw, section + 8, 1, 0x1000, 512, 512)
    struct.pack_into('<I', raw, section + 36, 0x60000020)
    raw[512] = 0xc3
    return bytes(raw)


class PackageStateCLITest(unittest.TestCase):
    write = staticmethod(fixtures.PrepareStateCLITest.write)

    def setUp(self):
        fixtures.PrepareStateCLITest.setUp(self)
        self.workspace = self.output
        self.output = self.root / '.cache/package.zip'
        self.manifest = prepare.prepare(self.workspace, self.bootstrap, root=self.root)
        self.manifest_path = self.workspace / 'prepare-manifest.json'
        self.image = self.workspace / package.IMAGE
        self.image.write_bytes(pe_image())
        self.write(self.root / 'LICENSE', b'fixture license\r\nexact bytes\n')
        self.record_path = self.root / '.cache/build.json'
        self.record = {
            'argv': self.manifest['buildCommands'][0]['argv'], 'cwd': str(self.workspace),
            'exit': 0, 'stderr': '', 'stdout': 'fixture record, not a native execution\n',
            'seconds': 1.0, 'image': package.identity(self.image.read_bytes()),
        }
        self.save_record()

    def save_record(self):
        self.record_path.write_bytes(package.json_bytes(self.record))

    def save_manifest(self):
        self.manifest_path.write_bytes(package.json_bytes(self.manifest))

    def run_package(self, output=None):
        return package.package(self.workspace, self.record_path, output or self.output, root=self.root)

    def refused(self, pattern=None):
        with self.assertRaisesRegex((ValueError, OSError), pattern or '.'):
            self.run_package()
        self.assertFalse(self.output.exists())

    def test_exact_bytes_members_relative_metadata_hashes_and_reproducibility(self):
        before = prepare.inventory(self.root)
        self.record['stdout'] = f'local compiler log: {self.root}\n'
        self.record['extraLocalPath'] = str(self.root / 'must-not-leak')
        self.save_record()
        with mock.patch('subprocess.run', side_effect=AssertionError('no execution')), \
                mock.patch('subprocess.Popen', side_effect=AssertionError('no execution')), \
                mock.patch('socket.socket', side_effect=AssertionError('no network')):
            metadata = self.run_package()
            second = self.root / '.cache/repeat.zip'
            for path in (self.image, self.root / 'LICENSE',
                         *(self.workspace / 'build' / name for name in prepare.PACK_NAMES.values())):
                os.utime(path, (1000000000, 1000000000))
            self.run_package(second)
        self.assertEqual(self.output.read_bytes(), second.read_bytes())
        self.assertEqual(before, prepare.inventory(self.root))
        self.assertEqual(metadata['profile'], 'low')
        with zipfile.ZipFile(self.output) as archive:
            expected = {'fairyc-state.exe', *prepare.PACK_NAMES.values(), 'LICENSE',
                        'README.md', 'metadata.json', 'SHA256SUMS'}
            self.assertEqual(set(archive.namelist()), expected)
            self.assertEqual(archive.namelist(), sorted(expected))
            for name in ('fairyc-state.exe', *prepare.PACK_NAMES.values()):
                self.assertEqual(archive.read(name), (self.workspace / 'build' / name).read_bytes())
            self.assertEqual(archive.read('LICENSE'), (self.root / 'LICENSE').read_bytes())
            self.assertEqual(archive.read('README.md'), package.README)
            self.assertEqual(json.loads(archive.read('metadata.json')), metadata)
            self.assertNotIn(str(self.root), archive.read('metadata.json').decode())
            self.assertNotIn(str(self.root).replace('\\', '\\\\'), archive.read('metadata.json').decode())
            self.assertNotIn(self.record['stdout'].encode(), archive.read('metadata.json'))
            for line in archive.read('SHA256SUMS').decode().splitlines():
                digest, name = line.split('  ')
                self.assertEqual(digest, prepare.digest(archive.read(name)))
            self.assertEqual(len(archive.read('SHA256SUMS').splitlines()), len(expected) - 1)
            for row in metadata['files']:
                self.assertEqual(package.identity(archive.read(row['path'])),
                                 {k: row[k] for k in ('bytes', 'sha256')})
            for info in archive.infolist():
                self.assertEqual(info.date_time, (1980, 1, 1, 0, 0, 0))
                self.assertEqual(info.compress_type, zipfile.ZIP_STORED)
                self.assertEqual(info.create_system, 3)
                self.assertEqual(info.external_attr >> 16, 0o100644)
                self.assertEqual(info.extra, b'')
                self.assertEqual(info.comment, b'')
            self.assertEqual(archive.comment, b'')

    def test_optimized_record_selects_optimized_metadata(self):
        self.record['argv'] = self.manifest['buildCommands'][1]['argv']
        self.record['profile'] = 'optimized'
        self.record['output'] = package.IMAGE
        self.save_record()
        self.assertEqual(self.run_package()['profile'], 'optimized')

    def test_manifest_tampering_schema_workspace_target_policy_and_inventories(self):
        original = copy.deepcopy(self.manifest)
        changes = {
            'schema': 'other', 'status': 'built', 'repository': 'elsewhere', 'workspace': 'elsewhere',
            'target': 'x86_64-linux-gnu', 'profiles': ['low'], 'profilesAreAlternatives': False,
            'entryDestination': '../main.fy', 'sourceRewrites': True, 'lockCopied': True,
            'stdlib': {}, 'inputs': self.manifest['inputs'][:-1],
            'copiedFiles': self.manifest['copiedFiles'][:-1], 'packs': [], 'buildCommands': [],
        }
        for key, value in changes.items():
            with self.subTest(key=key):
                self.manifest = {**copy.deepcopy(original), key: value}
                self.save_manifest()
                self.refused('mismatch')

    def test_stale_added_removed_and_changed_authored_sources(self):
        for path, replacement in ((self.entry, b'changed entry'),
                                  (self.source / 'src/nested/owner.fy', None),
                                  (self.source / 'src/added.fy', b'new file')):
            old = path.read_bytes() if path.exists() else None
            with self.subTest(path=path):
                if replacement is None:
                    path.unlink()
                else:
                    path.write_bytes(replacement)
                self.refused('authored inputs')
                if old is None:
                    path.unlink()
                else:
                    path.write_bytes(old)

    def test_tampered_copied_source_and_extra_source(self):
        path = self.workspace / 'src/main.fy'
        original = path.read_bytes()
        path.write_bytes(b'tampered')
        self.refused('copied src/main')
        path.write_bytes(original)
        (self.workspace / 'src/extra.fy').write_bytes(b'extra build input')
        self.refused('source tree')

    def test_bootstrap_hash_and_size_tampering(self):
        original = self.bootstrap.read_bytes()
        self.bootstrap.write_bytes(b'changed compiler')
        self.refused('bootstrap')
        self.bootstrap.write_bytes(original)
        self.manifest['bootstrap']['bytes'] += 1
        self.save_manifest()
        self.refused('bootstrap')

    def test_record_cwd_argv_profile_output_exit_stderr_and_image(self):
        original = copy.deepcopy(self.record)
        changes = [('cwd', str(self.root)), ('argv', original['argv'] + ['--locked']),
                   ('argv', original['argv'][:-1] + ['build/other.exe']),
                   ('argv', original['argv'][:6] + ['release'] + original['argv'][7:]),
                   ('profile', 'optimized'), ('output', 'other.exe'), ('exit', 1),
                   ('exit', False), ('stderr', 'warning'), ('stdout', None),
                   ('image', {**original['image'], 'bytes': 0}),
                   ('image', {**original['image'], 'sha256': '0' * 64})]
        for key, value in changes:
            with self.subTest(key=key, value=value):
                self.record = {**copy.deepcopy(original), key: value}
                self.save_record()
                self.refused()

    def test_record_and_prepared_command_cannot_jointly_redirect_build(self):
        self.manifest['buildCommands'][0]['argv'][-1] = 'build/other.exe'
        self.record['argv'] = self.manifest['buildCommands'][0]['argv']
        self.save_manifest()
        self.save_record()
        self.refused('prepared build commands')

    def test_real_image_bytes_must_match_record(self):
        self.image.write_bytes(self.image.read_bytes() + b'tampered')
        self.refused('image')

    def test_invalid_pe_images_even_with_matching_record_hash(self):
        original = pe_image()
        changes = [(b'not PE', None, None), (original, 0x3c, ('<I', 99999)),
                   (original, 0x84, ('<H', 0x14c)), (original, 0x98, ('<H', 0x10b)),
                   (original, 0x96, ('<H', 0x2002)), (original, 0x98 + 68, ('<H', 2)),
                   (original, 0x98 + 240 + 20, ('<I', 99999)),
                   (original, 0x98 + 16, ('<I', 0x4000))]
        for raw, offset, field in changes:
            with self.subTest(offset=offset, field=field):
                changed = bytearray(raw)
                if offset is not None:
                    struct.pack_into(field[0], changed, offset, field[1])
                self.image.write_bytes(changed)
                self.record['image'] = package.identity(changed)
                self.save_record()
                self.refused('PE')

    def test_pack_bytes_and_manifest_hash_tampering(self):
        for leaf in prepare.PACK_NAMES.values():
            path = self.workspace / 'build' / leaf
            raw = path.read_bytes()
            path.write_bytes(raw + b'x')
            self.refused('pack')
            path.write_bytes(raw)
        self.manifest['packs'][0]['sha256'] = '0' * 64
        self.save_manifest()
        self.refused('pack inventory')

    def test_live_packager_source_change_is_stale(self):
        path = self.root / 'tools/package-candidate-state-graph-runtime.py'
        path.write_bytes(path.read_bytes() + b'\n# changed owner\n')
        self.refused('authored inputs')

    def test_pin_drift_rejected_even_with_matching_inventory_and_copies(self):
        for kind, leaf in (('runtime', 'module_state_graph_native_asset_pin.fy'),
                           ('frame', 'module_state_graph_frame_pin.fy')):
            with self.subTest(kind=kind):
                source = self.source / 'src' / leaf
                copied = self.workspace / 'src' / leaf
                original = source.read_bytes()
                source.write_bytes(original + b'// changed pin\n')
                copied.write_bytes(source.read_bytes())
                for row in self.manifest['inputs']:
                    if row['path'].endswith('/' + leaf):
                        row.update(package.identity(source.read_bytes()))
                for row in self.manifest['copiedFiles']:
                    if row['destination'] == 'src/' + leaf:
                        row.update(package.identity(source.read_bytes()))
                self.save_manifest()
                self.refused('pin drift')
                source.write_bytes(original)
                copied.write_bytes(original)
                self.manifest = json.loads(self.manifest_path.read_bytes())
                for rows, key in ((self.manifest['inputs'], 'path'),
                                  (self.manifest['copiedFiles'], 'destination')):
                    for row in rows:
                        if row[key].endswith('/' + leaf):
                            row.update(package.identity(original))
                self.save_manifest()

    def test_existing_zip_and_file_not_overwritten(self):
        self.output.write_bytes(b'keep existing file')
        with self.assertRaises(FileExistsError):
            self.run_package()
        self.assertEqual(self.output.read_bytes(), b'keep existing file')

    def test_existing_valid_zip_not_overwritten(self):
        self.run_package()
        original = self.output.read_bytes()
        with self.assertRaises(FileExistsError):
            self.run_package()
        self.assertEqual(self.output.read_bytes(), original)

    def test_cache_escape_traversal_suffix_and_workspace_output_refused(self):
        for output in (self.root / 'outside.zip', self.root / '.cache',
                       self.root / '.cache/../escape.zip', self.root / '.cache/no-extension',
                       self.workspace / 'extra.zip'):
            with self.subTest(output=output), self.assertRaises(ValueError):
                self.run_package(output)
        outside = self.root / 'outside-workspace'
        outside.mkdir()
        with self.assertRaisesRegex(ValueError, 'strictly below'):
            package.package(outside, self.record_path, self.output, root=self.root)
        with self.assertRaisesRegex(ValueError, 'traversal'):
            package.package(self.workspace, self.root / '.cache/sub/../build.json',
                            self.output, root=self.root)

    def test_observed_reparse_paths_refused(self):
        original = Path.lstat
        for target in (self.output.parent, self.image, self.manifest_path, self.record_path,
                       self.workspace / 'src', self.bootstrap):
            def attributes(path, *args, **kwargs):
                if path == target:
                    return types.SimpleNamespace(st_mode=stat.S_IFDIR, st_file_attributes=0x400)
                return original(path, *args, **kwargs)
            with self.subTest(target=target), mock.patch.object(Path, 'lstat', attributes):
                self.refused('reparse')

    def test_real_symlink_output_and_input_refused(self):
        link = self.root / '.cache/linked'
        try:
            os.symlink(self.workspace, link, target_is_directory=True)
        except (OSError, NotImplementedError) as error:
            self.skipTest(f'OS does not permit test symlinks: {error}')
        with self.assertRaisesRegex(ValueError, 'links'):
            self.run_package(link / 'out.zip')
        self.image.unlink()
        os.symlink(self.bootstrap, self.image)
        self.refused('links')

    def test_changed_authored_snapshot_during_validation_refused(self):
        original = prepare.packs
        def changing(root, inputs):
            result = original(root, inputs)
            self.entry.write_bytes(b'changed during validation')
            return result
        with mock.patch.object(prepare, 'packs', side_effect=changing):
            self.refused('inputs changed')

    def test_changed_image_and_copy_during_validation_refused(self):
        original = prepare.inventory
        for target in (self.image, self.workspace / 'src/main.fy'):
            raw = target.read_bytes()
            count = 0
            def changing(root):
                nonlocal count
                count += 1
                result = original(root)
                if count == 2:
                    target.write_bytes(raw + b'changed during validation')
                return result
            with self.subTest(target=target), mock.patch.object(prepare, 'inventory', side_effect=changing):
                self.refused('changed')
            target.write_bytes(raw)

    def test_cli_has_required_flags_and_nonzero_refusal(self):
        with self.assertRaises(SystemExit) as caught, mock.patch('sys.stderr'):
            package.main(['--workspace', str(self.workspace), '--build-record', str(self.record_path),
                          '--output', str(self.root / 'outside.zip')])
        self.assertEqual(caught.exception.code, 2)


if __name__ == '__main__':
    unittest.main()
