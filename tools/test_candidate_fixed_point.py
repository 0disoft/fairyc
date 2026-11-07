"""Companion authentication for current and historical fixed-point inputs."""
import importlib.util
import io
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('fixed_point', Path(__file__).with_name('verify-candidate-native-fixed-point.py'))
fixed = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixed)

spec = importlib.util.spec_from_file_location('companion_fixed_point', Path(__file__).with_name('verify-candidate-companion-fixed-point.py'))
companion_fixed = importlib.util.module_from_spec(spec)
spec.loader.exec_module(companion_fixed)


class CompanionTests(unittest.TestCase):
    def fixture(self, root):
        def write(name, data):
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
            return path
        bootstrap = write('bootstrap.exe', b'bootstrap tool')
        seed = write('seed.exe', b'candidate seed')
        source = write('driver.fy', b'authored source')
        runtime = write('runtime.c', b'runtime source')
        write('driver/sources.list', b'driver.fy\n')
        packet = write('driver/driver.bin', b'FYCPARTS1\nopaque packet')
        provider = write('driver/release/fairyc-candidate-build.exe', b'authored driver')
        pack = write('driver/release/fairyc-runtime-assets.pack', b'packed runtime')
        image = write('cli/release/fairyc.exe', b'public compiler')
        write('cli/release/fairyc-candidate-build.exe', provider.read_bytes())
        installed_pack = write('cli/release/fairyc-runtime-assets.pack', pack.read_bytes())
        identity = lambda p: dict(path=str(p), sha256=fixed.sha(p))
        data = dict(schema='fairyc.candidate-authored-driver-build/v1', passed=True,
                    goFrontendInvoked=False, runtimeGoBuilderRequired=False, sourceFiles=1,
                    sources=[dict(path='driver.fy', sha256=fixed.sha(source))],
                    runtimeSources=[dict(path='runtime.c', sha256=fixed.sha(runtime))],
                    tools={'candidate': identity(seed), 'bootstrapBuilder': identity(bootstrap)},
                    images=[dict(profile='release', **identity(provider), assetPath=str(pack), assetSHA256=fixed.sha(pack))],
                    packetSHA256=fixed.sha(packet))
        receipt = write('driver/receipt.json', json.dumps(data).encode())
        built = dict(tools={'builder': identity(bootstrap), 'authoredDriverReceipt': identity(receipt)})
        release = dict(profile='release', path=str(image), driverImplementation='authored Fairy',
                       driverSHA256=fixed.sha(provider), assetPath=str(installed_pack), assetSHA256=fixed.sha(pack))
        return built, release

    def test_authored_companion_does_not_have_to_equal_bootstrap_builder(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            built, release = self.fixture(root)
            checked = fixed.checked_companion(built, release, root)
            self.assertNotEqual(checked['driver']['sha256'], built['tools']['builder']['sha256'])
            self.assertEqual(checked['assets']['sha256'], release['assetSHA256'])

    def test_changed_dependency_is_rejected(self):
        for name in ('cli/release/fairyc-candidate-build.exe', 'cli/release/fairyc-runtime-assets.pack',
                     'driver.fy', 'runtime.c', 'driver/receipt.json', 'driver/driver.bin',
                     'driver/sources.list', 'driver/release/fairyc-candidate-build.exe', 'bootstrap.exe'):
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                built, release = self.fixture(root)
                (root / name).write_bytes(b'changed')
                with self.assertRaises(AssertionError):
                    fixed.checked_companion(built, release, root)

    def test_authored_receipt_cannot_be_downgraded_to_legacy(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            built, release = self.fixture(root)
            del release['driverImplementation']
            with self.assertRaises(AssertionError):
                fixed.checked_companion(built, release, root)

    def test_historical_builder_companion_remains_accepted(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            built, release = self.fixture(root)
            del built['tools']['authoredDriverReceipt']
            del release['driverImplementation']
            driver = Path(release['path']).with_name('fairyc-candidate-build.exe')
            driver.write_bytes(Path(built['tools']['builder']['path']).read_bytes())
            release['driverSHA256'] = fixed.sha(driver)
            checked = fixed.checked_companion(built, release, root)
            self.assertEqual(checked['implementation'], 'historical bootstrap builder')


class CompilerHostTests(unittest.TestCase):
    def test_release_build_explicitly_selects_compiler_host(self):
        with tempfile.TemporaryDirectory() as directory:
            packet = Path(directory) / 'packet.bin'
            packet.write_bytes(b'FYCPARTS1\npacket')
            for reserve in (0, 16 << 20):
                with self.subTest(reserve=reserve):
                    expected = ['builder.exe', '--packet', str(packet), '--packet-sha256', fixed.sha(packet),
                                '--output', 'stage.exe', '--compiler-host', '--release']
                    if reserve:
                        expected += ['--stack-reserve-bytes', str(reserve)]
                    self.assertEqual(fixed.release_build_argv('builder.exe', packet, 'stage.exe', reserve), expected)


class ReceiptGateTests(unittest.TestCase):
    fixture = CompanionTests.fixture

    def build_fixture(self, root):
        built, release = self.fixture(root)
        release['sha256'] = fixed.sha(Path(release['path']))
        lock = root / 'examples/lexer_candidate/fairy.lock'
        lock.parent.mkdir(parents=True)
        lock.write_bytes(b'frozen project lock')
        packet = root / 'cli/candidate-packet.bin'
        packet.write_bytes((root / 'driver/driver.bin').read_bytes())
        manifest = root / 'cli/sources.list'
        manifest.write_text('driver.fy\n', encoding='utf-8')
        seed = root / 'seed.exe'
        source = dict(path='driver.fy', sha256=fixed.sha(root / 'driver.fy'))
        built.update(schema='fairyc.candidate-native-probe-build/v1', passed=True,
                     goFrontendInvoked=False, commandsOnly=False, compilerOnly=False,
                     frontendReuse=None, sources=[source], lockSHA256=fixed.sha(lock),
                     images=[release], stackReserveBytes=16 << 20,
                     candidatePacketSHA256=fixed.sha(packet),
                     buildVersion=dict(text='frozen version\n', template=source,
                                       oracle=dict(path=str(seed), sha256=fixed.sha(seed))))
        built['tools']['candidate'] = dict(path=str(seed), sha256=fixed.sha(seed))
        built['invocations'] = [dict(argv=[str(seed), '--c-project-file', str(manifest)],
                                    exit=0, stderr='', stdoutSHA256=fixed.sha(packet),
                                    stdoutBytes=packet.stat().st_size),
                                dict(argv=fixed.release_build_argv(root / 'bootstrap.exe', packet,
                                                                  Path(release['path']), 16 << 20),
                                     exit=0, stderr='')]
        path = root / 'cli/receipt.json'
        path.write_text(json.dumps(built), encoding='utf-8')
        return path, lock

    def verify(self, verifier, root, receipt, lock, drift=False):
        """Run each real verifier entry point with only native processes faked."""
        proof = verifier if verifier is fixed else verifier.proof
        source_check, lock_check = proof.checked_source_inputs, proof.checked_lock
        companion_check = proof.checked_companion
        output = root / '.cache/verified'
        calls = []

        def process(argv, **kwargs):
            calls.append(argv)
            if '--c-project-file' in argv:
                out = (root / 'driver/driver.bin').read_bytes()
            elif '--version' in argv:
                out = b'frozen version\n'
            else:
                self.assertIn('--output', argv)
                target = Path(argv[argv.index('--output')+1])
                if '--output-root' in argv:
                    target = Path(argv[argv.index('--output-root')+1]) / target
                original = root / ('driver/release/fairyc-candidate-build.exe'
                                   if '--packet-input' in argv else 'cli/release/fairyc.exe')
                target.write_bytes(original.read_bytes())
                out = b''

            def communicate(timeout):
                if drift and len(calls) == 3:
                    lock.write_bytes(b'changed during verification')
                return out, b''
            return SimpleNamespace(returncode=0, communicate=communicate)

        with patch.object(verifier, 'ROOT', root), \
                patch.object(proof, 'checked_source_inputs', side_effect=lambda rows: source_check(rows, root)), \
                patch.object(proof, 'checked_lock', side_effect=lambda built: lock_check(built, root)), \
                patch.object(proof, 'checked_companion', side_effect=lambda built, release: companion_check(built, release, root)), \
                patch.object(proof, 'stack_reserve', return_value=16 << 20), \
                patch.object(verifier.subprocess, 'Popen', side_effect=process) as popen, \
                patch('sys.argv', [verifier.__file__, '--build-receipt', str(receipt), '--output', str(output)]), \
                patch('sys.stdout', new_callable=io.StringIO):
            try:
                verifier.main()
            finally:
                self.process_calls = popen.call_count
        return json.loads((output / 'receipt.json').read_text(encoding='utf-8'))

    def test_unchanged_inputs_keep_both_success_receipt_formats(self):
        for verifier, schema in ((fixed, 'fairyc.candidate-native-cli-fixed-point/v1'),
                                 (companion_fixed, 'fairyc.candidate-companion-fixed-point/v1')):
            with self.subTest(verifier=verifier.__name__), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                receipt, lock = self.build_fixture(root)
                result = self.verify(verifier, root, receipt, lock)
                self.assertEqual(result['schema'], schema)
                for key in ('passed', 'packetBytesEqual', 'optimizedImageBytesEqual'):
                    self.assertIs(result[key], True)
                self.assertEqual(result['excludedImageBytes'], [])
                self.assertEqual(self.process_calls, 3)

    def test_empty_sources_fail_before_any_process(self):
        for verifier in (fixed, companion_fixed):
            with self.subTest(verifier=verifier.__name__), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                receipt, lock = self.build_fixture(root)
                built = json.loads(receipt.read_text(encoding='utf-8'))
                built['sources'] = []
                receipt.write_text(json.dumps(built), encoding='utf-8')
                with self.assertRaisesRegex(AssertionError, 'source file count'):
                    self.verify(verifier, root, receipt, lock)
                self.assertEqual(self.process_calls, 0)
                self.assertFalse((root / '.cache/verified/receipt.json').exists())

    def test_lock_drift_rejected_before_start_and_before_success(self):
        for verifier in (fixed, companion_fixed):
            for when in ('before_start', 'before_success'):
                with self.subTest(verifier=verifier.__name__, when=when), tempfile.TemporaryDirectory() as directory:
                    root = Path(directory)
                    receipt, lock = self.build_fixture(root)
                    if when == 'before_start':
                        lock.write_bytes(b'changed before verification')
                    with self.assertRaisesRegex(AssertionError, 'compiler project lock changed'):
                        self.verify(verifier, root, receipt, lock, drift=when == 'before_success')
                    self.assertEqual(self.process_calls, 0 if when == 'before_start' else 3)
                    self.assertFalse((root / '.cache/verified/receipt.json').exists())


if __name__ == '__main__':
    unittest.main()
