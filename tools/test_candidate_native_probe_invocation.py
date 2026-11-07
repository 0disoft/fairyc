"""Fast native-probe logging checks; subprocesses and clocks are injected."""
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import Mock, patch

from candidate_native_probe_invocation import run_invocation


class InvocationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.output = Path(self.directory.name)
        self.cwd = self.output / 'working-directory'
        self.argv = [Path('candidate.exe'), '--c-project-file', Path('sources.list')]
        self.calls = []
        self.clock = Mock(side_effect=[10.0, 12.5])

    def invoke(self, runner, timeout=900):
        return run_invocation(self.argv, timeout, cwd=self.cwd, output=self.output,
                              calls=self.calls, runner=runner, clock=self.clock)

    def log(self):
        rows = json.loads((self.output / 'invocations.json').read_text(encoding='utf-8'))
        self.assertEqual(rows, self.calls)
        self.assertEqual(self.clock.call_count, 2)
        self.assertFalse(list(self.output.glob('receipt*.json')))
        return rows[-1]

    def assert_output_files(self, row, stdout, stderr):
        self.assertEqual((self.output / row['stdoutPath']).read_bytes(), stdout)
        self.assertEqual((self.output / row['stderrPath']).read_bytes(), stderr)
        self.assertEqual(row['stdoutBytes'], len(stdout))
        self.assertEqual(row['stdoutSHA256'], hashlib.sha256(stdout).hexdigest())
        self.assertEqual(row['stderr'], stderr.decode('utf-8', errors='replace'))

    def test_timeout_preserves_partial_binary_output_and_original_exception(self):
        stdout, stderr = b'FYCPARTS1\n\x00\xffpartial', b'phase: emit\n\xfe'
        error = subprocess.TimeoutExpired(self.argv, 900, output=stdout, stderr=stderr)
        runner = Mock(side_effect=error)
        # Retain prior evidence and number failure artifacts after earlier calls.
        self.calls.append(dict(argv=['earlier'], exit=0))
        with self.assertRaises(subprocess.TimeoutExpired) as caught:
            self.invoke(runner)
        self.assertIs(caught.exception, error)
        row = self.log()
        self.assertEqual(len(self.calls), 2)
        self.assertEqual(row['stdoutPath'], 'invocation-002.stdout.bin')
        self.assert_output_files(row, stdout, stderr)
        self.assertIsNone(row['exit'])
        self.assertEqual(row['seconds'], 2.5)
        self.assertTrue(row['timedOut'])
        self.assertEqual(row['timeoutSeconds'], 900)
        self.assertEqual(row['error'], dict(type='TimeoutExpired', message=str(error)))
        runner.assert_called_once_with(list(map(str, self.argv)), cwd=self.cwd,
                                       capture_output=True, timeout=900)

    def test_nonzero_exit_records_output_and_still_fails(self):
        runner = Mock(return_value=subprocess.CompletedProcess(self.argv, 7, b'partial\x00', b'failed\n'))
        with self.assertRaises(AssertionError) as caught:
            self.invoke(runner)
        row = self.log()
        self.assertEqual(caught.exception.args, (row,))
        self.assertEqual(row['exit'], 7)
        self.assertNotIn('error', row)
        self.assert_output_files(row, b'partial\x00', b'failed\n')

    def test_zero_exit_with_stderr_still_fails(self):
        runner = Mock(return_value=subprocess.CompletedProcess(self.argv, 0, b'', b'warning'))
        with self.assertRaises(AssertionError):
            self.invoke(runner)
        self.assert_output_files(self.log(), b'', b'warning')

    def test_timeout_without_output_records_empty_artifacts(self):
        error = subprocess.TimeoutExpired(self.argv, 30)
        with self.assertRaises(subprocess.TimeoutExpired) as caught:
            self.invoke(Mock(side_effect=error), timeout=30)
        self.assertIs(caught.exception, error)
        self.assert_output_files(self.log(), b'', b'')

    def test_success_keeps_exact_schema_bytes_hash_and_elapsed_time(self):
        stdout = b'FYCPARTS1\n\x00\xffpacket'
        self.assertEqual(self.invoke(Mock(return_value=subprocess.CompletedProcess(
            self.argv, 0, stdout, b''))), stdout)
        self.assertEqual(self.log(), dict(argv=list(map(str, self.argv)), seconds=2.5,
                         exit=0, stdoutBytes=len(stdout),
                         stdoutSHA256=hashlib.sha256(stdout).hexdigest(), stderr=''))
        self.assertEqual([p.name for p in self.output.iterdir()], ['invocations.json'])

    def test_empty_success_keeps_empty_hash(self):
        self.assertEqual(self.invoke(Mock(return_value=subprocess.CompletedProcess(
            self.argv, 0, b'', b''))), b'')
        row = self.log()
        self.assertEqual(row['stdoutBytes'], 0)
        self.assertEqual(row['stdoutSHA256'], hashlib.sha256(b'').hexdigest())

    def test_launch_error_is_logged_and_propagated_unchanged(self):
        error = FileNotFoundError('candidate unavailable')
        with self.assertRaises(FileNotFoundError) as caught:
            self.invoke(Mock(side_effect=error))
        self.assertIs(caught.exception, error)
        row = self.log()
        self.assertEqual(row['error'], dict(type='FileNotFoundError', message=str(error)))
        self.assertIsNone(row['exit'])
        self.assert_output_files(row, b'', b'')

    def test_called_process_error_preserves_captured_output(self):
        error = subprocess.CalledProcessError(3, self.argv, output=b'part', stderr=b'bad')
        with self.assertRaises(subprocess.CalledProcessError) as caught:
            self.invoke(Mock(side_effect=error))
        self.assertIs(caught.exception, error)
        row = self.log()
        self.assertEqual(row['exit'], 3)
        self.assert_output_files(row, b'part', b'bad')

    def test_diagnostic_write_error_does_not_replace_timeout(self):
        error = subprocess.TimeoutExpired(self.argv, 900, output=b'part')
        with patch.object(Path, 'write_bytes', side_effect=OSError('disk unavailable')):
            with self.assertRaises(subprocess.TimeoutExpired) as caught:
                self.invoke(Mock(side_effect=error))
        self.assertIs(caught.exception, error)


if __name__ == '__main__':
    unittest.main()
