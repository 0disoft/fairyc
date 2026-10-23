"""Companion authentication for current and historical fixed-point inputs."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('fixed_point', Path(__file__).with_name('verify-candidate-native-fixed-point.py'))
fixed = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixed)


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


if __name__ == '__main__':
    unittest.main()
