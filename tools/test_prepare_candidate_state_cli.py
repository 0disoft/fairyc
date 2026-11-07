"""Cheap isolated tests; no Go, C compiler or product execution."""
import importlib.util
import json
import os
from pathlib import Path
import stat
import tempfile
import types
import unittest
from unittest import mock

TOOLS = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('prepare_state_cli', TOOLS / 'prepare-candidate-state-cli.py')
prepare = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prepare)


class PrepareStateCLITest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.source = self.root / 'examples/lexer_candidate'
        self.entry = self.root / 'tools/dev/fairyc-state/main.fy'
        self.bootstrap = self.root / 'compiler with spaces.exe'
        self.bootstrap.write_bytes(b'not executed: explicit bootstrap identity')
        self.write(self.source / 'fairy.toml', b'schema = "fairyc.manifest/v1"\n[package]\nsource = "src"\n')
        self.write(self.source / 'src/main.fy', b'module old_main;\r\n')
        self.write(self.source / 'src/nested/owner.fy', b'module owner;\r\n// exact authored bytes\r\n')
        self.write(self.source / 'fairy.lock', b'must never be copied')
        self.write(self.entry, b'module main;\r\n// actual developer entry\r\n')
        # Use the real distribution-time packagers and their authoritative
        # inputs/pins in a tiny injected checkout. No fake execution/provider.
        for kind in prepare.PACK_NAMES:
            name = f'package-candidate-state-graph-{kind}.py'
            self.write(self.root / 'tools' / name, (TOOLS / name).read_bytes())
        for name in ('stage0_contract_state_overlay.h', 'stage0_contract_state_overlay.c',
                     'stage0_contract_state_graph.h', 'stage0_contract_state_graph.c',
                     'stage0_contract_state_graph_key.c'):
            self.write(self.root / 'internal/cstdlib' / name,
                       (TOOLS.parent / 'internal/cstdlib' / name).read_bytes())
        self.write(self.root / 'internal/contractstatenative/graph_native_c.go',
                   (TOOLS.parent / 'internal/contractstatenative/graph_native_c.go').read_bytes())
        for name in ('module_state_graph_native_asset_pin.fy', 'module_state_graph_frame_pin.fy'):
            self.write(self.source / 'src' / name,
                       (TOOLS.parent / 'examples/lexer_candidate/src' / name).read_bytes())
        self.output = self.root / '.cache/fresh/workspace'

    @staticmethod
    def write(path, raw):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(raw)

    def run_prepare(self, output=None):
        return prepare.prepare(output or self.output, self.bootstrap, root=self.root)

    def test_fresh_exact_copy_inventory_packs_and_commands(self):
        before = prepare.inventory(self.root)
        with mock.patch('subprocess.run', side_effect=AssertionError('must not execute any build')):
            manifest = self.run_prepare()
        self.assertEqual(before, prepare.inventory(self.root))
        self.assertEqual((self.output / 'src/main.fy').read_bytes(), self.entry.read_bytes())
        self.assertEqual((self.output / 'src/nested/owner.fy').read_bytes(),
                         (self.source / 'src/nested/owner.fy').read_bytes())
        self.assertEqual((self.output / 'fairy.toml').read_bytes(), (self.source / 'fairy.toml').read_bytes())
        self.assertFalse((self.output / 'fairy.lock').exists())
        self.assertFalse((self.root / 'tools/__pycache__').exists())
        self.assertEqual({p.name for p in (self.output / 'build').iterdir()}, set(prepare.PACK_NAMES.values()))
        for pack in manifest['packs']:
            raw = (self.output / pack['path']).read_bytes()
            self.assertEqual(pack['sha256'], prepare.digest(raw))
            self.assertEqual(pack['bytes'], len(raw))
        self.assertEqual(manifest['status'], 'prepared-not-built')
        self.assertFalse(manifest['stdlib']['nativeBuildVerified'])
        for profile, command in zip(('low', 'optimized'), manifest['buildCommands']):
            self.assertEqual(command['cwd'], str(self.output))
            self.assertFalse(command['executed'])
            self.assertEqual(command['argv'], [str(self.bootstrap), 'experimental', 'build-workspace',
                '--target', 'x86_64-windows-msvc', '--profile', profile,
                '--output', 'build/fairyc-state.exe'])
            image_directory = self.output / Path(command['argv'][-1]).parent
            self.assertTrue(all((image_directory / leaf).is_file() for leaf in prepare.PACK_NAMES.values()))
        for copied in manifest['copiedFiles']:
            self.assertEqual(copied['sha256'], prepare.digest(before[copied['source']]))
            self.assertEqual((self.output / copied['destination']).read_bytes(), before[copied['source']])
        self.assertEqual(json.loads((self.output / 'prepare-manifest.json').read_bytes()), manifest)

    def test_reuse_refused_without_overwriting(self):
        self.run_prepare()
        before = {p.relative_to(self.output): p.read_bytes() for p in self.output.rglob('*') if p.is_file()}
        with self.assertRaises(FileExistsError):
            self.run_prepare()
        self.assertEqual(before, {p.relative_to(self.output): p.read_bytes() for p in self.output.rglob('*') if p.is_file()})

    def test_output_escape_source_overwrite_cache_root_and_traversal_refused(self):
        before = prepare.inventory(self.root)
        for output in (self.root / 'outside', self.source, self.root / '.cache',
                       self.root / '.cache/../escape'):
            with self.subTest(output=output), self.assertRaises(ValueError):
                self.run_prepare(output)
        self.assertEqual(before, prepare.inventory(self.root))
        self.assertFalse((self.root / 'outside').exists())

    def test_existing_file_output_refused(self):
        self.output.parent.mkdir(parents=True)
        self.output.write_bytes(b'keep')
        with self.assertRaises(FileExistsError):
            self.run_prepare()
        self.assertEqual(self.output.read_bytes(), b'keep')

    def test_pin_drift_refused_before_creating_output(self):
        pin = self.source / 'src/module_state_graph_frame_pin.fy'
        pin.write_bytes(pin.read_bytes() + b'// drift\n')
        with self.assertRaisesRegex(ValueError, 'pin drift'):
            self.run_prepare()
        self.assertFalse(self.output.exists())

    def test_manifest_source_and_dependency_changes_require_review(self):
        path = self.source / 'fairy.toml'
        for raw in (b'schema="fairyc.manifest/v1"\n[package]\nsource="../elsewhere"\n',
                    b'schema="fairyc.manifest/v1"\n[package]\nsource="src"\n[dependencies]\n'):
            path.write_bytes(raw)
            with self.assertRaises(ValueError):
                self.run_prepare()
            self.assertFalse(self.output.exists())

    def test_source_change_is_not_claimed_as_a_successful_snapshot(self):
        original = prepare.packs
        def changing(root, inputs):
            result = original(root, inputs)
            self.entry.write_bytes(b'changed by another worker')
            return result
        with mock.patch.object(prepare, 'packs', side_effect=changing):
            with self.assertRaisesRegex(ValueError, 'inputs changed'):
                self.run_prepare()
        self.assertTrue(self.output.exists())  # Never delete somebody's evidence.
        self.assertFalse((self.output / 'prepare-manifest.json').exists())

    def test_junction_reparse_attribute_rejected_before_resolving(self):
        junction = self.root / '.cache/junction'
        original = Path.lstat
        def attributes(path, *args, **kwargs):
            if path == junction:
                return types.SimpleNamespace(st_mode=stat.S_IFDIR, st_file_attributes=0x400)
            return original(path, *args, **kwargs)
        with mock.patch.object(Path, 'lstat', attributes):
            with self.assertRaisesRegex(ValueError, 'reparse'):
                prepare.no_links(junction / 'new')

    def test_real_symlink_source_and_output_refused(self):
        link = self.root / 'link'
        try:
            os.symlink(self.source, link, target_is_directory=True)
        except (OSError, NotImplementedError) as error:
            self.skipTest(f'OS does not permit creating test symlinks: {error}')
        with self.assertRaisesRegex(ValueError, 'links'):
            prepare.no_links(link / 'src/main.fy')
        (self.root / '.cache').mkdir()
        os.symlink(self.source, self.root / '.cache/linked', target_is_directory=True)
        with self.assertRaisesRegex(ValueError, 'links'):
            self.run_prepare(self.root / '.cache/linked/workspace')
        os.symlink(self.entry, self.source / 'src/linked.fy')
        with self.assertRaisesRegex(ValueError, 'links'):
            self.run_prepare()


if __name__ == '__main__':
    unittest.main()
