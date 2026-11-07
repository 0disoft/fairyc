"""Source admission and bootstrap argv regressions; no native build is launched."""
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(filename))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


probe = load('native_probe', 'build-candidate-native-probe.py')
fixed = load('native_fixed_point', 'verify-candidate-native-fixed-point.py')


class SourceAdmissionTests(unittest.TestCase):
    def check(self, root, paths, accepted):
        rows = [dict(path=p.relative_to(root).as_posix(), sha256=probe.sha(p), bytes=p.stat().st_size)
                for p in paths]
        for label, check in (
                ('probe', lambda: probe.source_snapshot(paths, root)),
                ('fixed-point', lambda: fixed.checked_source_inputs(rows, root))):
            with self.subTest(consumer=label, files=len(paths), accepted=accepted):
                if accepted:
                    check()
                else:
                    with self.assertRaises(AssertionError):
                        check()

    def test_limits_match_authored_owner(self):
        text = (probe.PROJECT / 'src/candidate_limits.fy').read_text(encoding='utf-8')
        for function, attribute in [('source_files', 'MAX_SOURCE_FILES'), ('source_bytes', 'MAX_SOURCE_BYTES'),
                                    ('source_set_bytes', 'MAX_SOURCE_SET_BYTES')]:
            value = int(re.search(r'pub fn usize ' + function + r'\(\)\{return (\d+);\}', text)[1])
            self.assertEqual(getattr(probe, attribute), value)
            self.assertEqual(getattr(fixed, attribute), value)

    def test_file_count_exact_and_one_over(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            paths = []
            for i in range(1025):
                path = root / f'm{i}.fy'
                path.write_bytes(f'module m{i};'.encode())
                paths.append(path)
            self.check(root, [], False)
            self.check(root, paths[:646], True)
            self.check(root, paths[:1024], True)
            self.check(root, paths, False)

    def test_source_set_counts_separators_and_keeps_per_file_limit(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            paths = []
            for i in range(128):
                path = root / f'm{i}.fy'
                size = 65536 if i < 127 else 65536 - 127
                path.write_bytes(b' ' * size)
                paths.append(path)
            self.check(root, paths, True)  # Exactly 8 MiB including 127 separators.
            paths[-1].write_bytes(paths[-1].read_bytes() + b' ')
            self.check(root, paths, False)
            paths[-1].write_bytes(b' ' * 65537)
            self.check(root, paths[-1:], False)

    def test_fixed_point_rejects_changed_source(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = root / 'main.fy'
            path.write_bytes(b'module main;')
            rows = probe.source_snapshot([path], root)
            path.write_bytes(b'module changed;')
            with self.assertRaisesRegex(AssertionError, 'source changed'):
                fixed.checked_source_inputs(rows, root)


class ProbeArgvTests(unittest.TestCase):
    def test_full_and_bounded_entries_select_compiler_host(self):
        for mode in ([], ['--compiler-only'], ['--commands-only']):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                project = root / 'examples/lexer_candidate'
                src = project / 'src'
                src.mkdir(parents=True)
                (project / 'fairy.lock').write_bytes(b'lock')
                for name in ('file_cli', 'project_check_cli', 'native_cli', 'module_project_cli', 'public_cli_data'):
                    (src / f'{name}.fy').write_text(f'module {name};\n')
                for name in ('bytes', 'process'):
                    path = root / 'internal/sdkstdlib/sources/std' / f'{name}.fy'
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(f'module std::{name};\n')
                seed, builder = root / 'seed.exe', root / 'builder.exe'
                seed.write_bytes(b'seed')
                builder.write_bytes(b'builder')
                driver, assets = root / 'driver.exe', root / 'assets.pack'
                driver.write_bytes(b'authored driver')
                assets.write_bytes(b'runtime assets')
                receipt = root / 'driver-receipt.json'
                receipt.write_text(json.dumps(dict(schema='fairyc.candidate-authored-driver-build/v1', passed=True,
                    runtimeGoBuilderRequired=False, sources=[], runtimeSources=[], images=[dict(profile='release',
                        path=str(driver), sha256=probe.sha(driver), assetPath=str(assets), assetSHA256=probe.sha(assets))])))
                output = root / '.cache/probe'
                version = b'current source version\n'
                calls = []

                def generate(destination):
                    path = destination / 'public_cli_data.fy'
                    path.write_text('module public_cli_data;\n')
                    metadata = dict(text=version.decode(),
                        template=dict(path=(src / 'public_cli_data.fy').relative_to(root).as_posix(),
                                      sha256=probe.sha(src / 'public_cli_data.fy')),
                        oracle=dict(path=str(builder), sha256=probe.sha(builder)),
                        generated=dict(path=path.relative_to(root).as_posix(), sha256=probe.sha(path)))
                    return path, metadata

                def run(argv, **kwargs):
                    argv = list(map(str, argv))
                    calls.append(argv)
                    if argv[0] == str(seed):
                        return subprocess.CompletedProcess(argv, 0, b'FYCPARTS1\npacket', b'')
                    if argv[0] == str(builder):
                        Path(argv[argv.index('--output') + 1]).write_bytes(b'image')
                        return subprocess.CompletedProcess(argv, 0, b'built', b'')
                    self.assertEqual(argv[1:], ['--version'])
                    return subprocess.CompletedProcess(argv, 0, version, b'')

                snapshot = probe.source_snapshot
                argv = ['probe', '--candidate', str(seed), '--builder', str(builder), '--output', str(output),
                        '--authored-driver-receipt', str(receipt), '--profile', 'release', '--stack-reserve-bytes', '0', *mode]
                with patch.object(probe, 'ROOT', root), patch.object(probe, 'PROJECT', project), \
                        patch.object(probe, 'source_snapshot', lambda paths: snapshot(paths, root)), \
                        patch.object(probe, 'generate_version', generate), patch.object(probe.subprocess, 'run', run), \
                        patch.object(sys, 'argv', argv):
                    probe.main()
                built = next(call for call in calls if call[0] == str(builder))
                self.assertEqual(built, fixed.release_build_argv(builder, output / 'candidate-packet.bin',
                                                               output / 'release/fairyc.exe', 0))
                entry = (output / 'main.fy').read_text()
                if not mode:
                    self.assertIn('@effects(filesystem,console,environment,network,clock)', entry)
                else:
                    self.assertNotIn('network', entry)


if __name__ == '__main__':
    unittest.main()
