"""Portable fake-process tests. Passing here never means Linux native passed."""

import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location(
    'linux_collector', Path(__file__).with_name('collect-linux-native-validation.py'))
collector = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(collector)


def events(root, action='pass', **extra):
    return json.dumps({'Test': root, 'Action': action, **extra}) + '\n'


class CollectorTests(unittest.TestCase):
    def test_actual_source_roots_and_bounded_default(self):
        plan = collector.make_plan(collector.ROOT)
        self.assertEqual(plan['groups'], ['candidatebuild', 'trap-runtime'])
        self.assertNotIn('cli', plan['groups'])
        self.assertEqual(plan['requiredOptIns'], {})
        all_groups = collector.make_plan(collector.ROOT, list(collector.GROUPS))
        for command in all_groups['commands']:
            self.assertNotIn('./...', command['argv'])
            self.assertTrue(command['argv'][-1].startswith('^('))
            self.assertTrue(command['argv'][-1].endswith(')$'))
            self.assertEqual(set(command['roots']), set(command['sources']))

    def test_selection_replaces_defaults_and_deduplicates(self):
        plan = collector.make_plan(collector.ROOT, ['cli', 'cli'], 120)
        self.assertEqual(plan['groups'], ['cli'])
        self.assertEqual(plan['commands'][0]['roots'], ['TestLinuxFileCLI'])
        self.assertIn('120s', plan['commands'][0]['argv'])
        self.assertEqual(plan['commands'][0]['timeoutSeconds'], 150)
        for selection in ([], ['unknown']):
            with self.assertRaises(ValueError):
                collector.make_plan(collector.ROOT, selection)

    def test_missing_source_root_refused_without_running_go(self):
        with tempfile.TemporaryDirectory() as home:
            with self.assertRaisesRegex(ValueError, 'missing or ambiguous'):
                collector.make_plan(Path(home), ['cli'])

    def test_default_plan_has_no_subprocess_or_compiler_invocation(self):
        with tempfile.TemporaryDirectory() as home, \
                mock.patch.object(collector.subprocess, 'Popen') as popen, \
                mock.patch.object(collector, 'collect') as collect, \
                contextlib.redirect_stdout(io.StringIO()):
            output = Path(home) / 'plan'
            self.assertEqual(collector.main(['--output', str(output)]), 0)
            popen.assert_not_called()
            collect.assert_not_called()
            self.assertEqual(sorted(path.name for path in output.iterdir()), ['plan.json'])
            plan = json.loads((output / 'plan.json').read_text())
            self.assertEqual(plan['mode'], 'plan')
            self.assertEqual(plan['environmentOverrides']['GOTOOLCHAIN'], 'local')
            with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
                collector.main(['--output', str(output)])

    def test_windows_execute_rejected_before_output_or_process(self):
        with tempfile.TemporaryDirectory() as home, \
                mock.patch.object(collector.sys, 'platform', 'win32'), \
                mock.patch.object(collector.subprocess, 'Popen') as popen, \
                contextlib.redirect_stderr(io.StringIO()):
            output = Path(home) / 'run'
            with self.assertRaises(SystemExit):
                collector.main(['--execute', '--output', str(output)])
            self.assertFalse(output.exists())
            popen.assert_not_called()

    def test_environment_downloads_disabled_and_helper_modes_removed(self):
        with mock.patch.dict(collector.os.environ, {
                'GOTOOLCHAIN': 'auto', 'GOPROXY': 'https://example.invalid',
                'GOSUMDB': 'example.invalid', 'GOFLAGS': '-mod=mod',
                'FAIRYC_RUN_PARITY_INTEGRATION': '1', 'FAIRYC_HELPER_SECRET': 'secret'}):
            env = collector.child_environment()
        for key, value in collector.ENVIRONMENT.items():
            self.assertEqual(env[key], value)
        self.assertFalse(any(key.startswith('FAIRYC_') for key in env))

    def test_root_skip_partial_missing_failure_and_invalid_json(self):
        with tempfile.TemporaryDirectory() as home:
            path = Path(home) / 'stdout'
            path.write_text(events('TestA', 'skip', Output='requires Linux') +
                            events('TestB/compiler', 'skip') + events('TestB') +
                            events('TestD', 'fail') + 'not json\n', encoding='utf-8')
            summary = collector.summarize_tests(path, ['TestA', 'TestB', 'TestC', 'TestD'])
            self.assertEqual([r['status'] for r in summary['roots'].values()],
                             ['skip', 'partial', 'missing', 'fail'])
            self.assertEqual(summary['roots']['TestB']['skips'], ['TestB/compiler'])
            self.assertTrue(summary['packageFailed'])
            self.assertFalse(summary['complete'])
            self.assertEqual(len(summary['parseErrors']), 1)

    def test_duplicate_terminal_event_is_incomplete(self):
        with tempfile.TemporaryDirectory() as home:
            path = Path(home) / 'stdout'
            path.write_text(events('TestA') * 2, encoding='utf-8')
            result = collector.summarize_tests(path, ['TestA'])
            self.assertFalse(result['complete'])
            self.assertIn('duplicate', result['parseErrors'][0])

    def test_malformed_event_shapes_are_recorded_without_crashing(self):
        malformed = [[], None, 7, {'Action': 'pass', 'Test': []},
                     {'Action': [], 'Test': 'TestA'}, {'Action': None},
                     {'Action': 'pass', 'Test': None},
                     {'Action': 'output', 'Output': {}},
                     {'Action': 'pass', 'Package': []}]
        with tempfile.TemporaryDirectory() as home:
            path = Path(home) / 'stdout'
            path.write_text(''.join(json.dumps(event) + '\n' for event in malformed) +
                            events('TestA'), encoding='utf-8')
            result = collector.summarize_tests(path, ['TestA'])
            self.assertEqual(len(result['parseErrors']), len(malformed))
            self.assertEqual(result['roots']['TestA']['status'], 'pass')
            self.assertFalse(result['complete'])

    def fake_runner(self, plan, codes=None, skipped=(), interrupted=()):
        commands = {command['group']: command for command in plan['commands']}

        def run(argv, cwd, env, output, name, timeout):
            command = commands.get(name)
            text = ''.join(events(root, 'skip' if name in skipped else 'pass')
                           for root in command['roots']) if command else 'fake tool version\n'
            (output / (name + '.stdout')).write_text(text, encoding='utf-8')
            (output / (name + '.stderr')).write_text('fake stderr\n', encoding='utf-8')
            return {'argv': argv, 'cwd': str(cwd), 'startedAt': '2026-11-07T00:00:00+00:00',
                    'durationSeconds': 0.25, 'exitCode': (codes or {}).get(name, 0),
                    'effectiveExitCode': (codes or {}).get(name, 0),
                    'stdout': name + '.stdout', 'stderr': name + '.stderr',
                    'interrupted': name in interrupted, 'timedOut': False}
        return run

    def test_fake_collection_preserves_argv_logs_failures_and_continues(self):
        plan = collector.make_plan(collector.ROOT)
        with tempfile.TemporaryDirectory() as home, \
                mock.patch.dict(collector.os.environ, {'VERY_PRIVATE_TOKEN': 'never-record-me'}):
            output = Path(home)
            code = collector.collect(plan, output, self.fake_runner(plan, {'candidatebuild': 7}))
            self.assertEqual(code, 7)
            text = (output / 'receipt.json').read_text(encoding='utf-8')
            receipt = json.loads(text)
            self.assertNotIn('never-record-me', text)
            self.assertNotIn('VERY_PRIVATE_TOKEN', text)
            self.assertFalse(receipt['complete'])
            self.assertEqual(len(receipt['commands']), 2)
            first = receipt['commands'][0]
            self.assertEqual(first['argv'], plan['commands'][0]['argv'])
            self.assertEqual(first['exitCode'], 7)
            self.assertEqual(first['durationSeconds'], 0.25)
            self.assertEqual((output / first['stderr']).read_text(), 'fake stderr\n')

    def test_fake_skips_are_incomplete_and_success_is_complete(self):
        plan = collector.make_plan(collector.ROOT, ['candidatebuild'])
        for skipped, expected in ((['candidatebuild'], 3), ([], 0)):
            with self.subTest(skipped=skipped), tempfile.TemporaryDirectory() as home:
                output = Path(home)
                code = collector.collect(plan, output, self.fake_runner(plan, skipped=skipped))
                receipt = json.loads((output / 'receipt.json').read_text())
                self.assertEqual(code, expected)
                self.assertEqual(receipt['complete'], expected == 0)

    def test_fake_interruption_records_unrun_selection(self):
        plan = collector.make_plan(collector.ROOT)
        with tempfile.TemporaryDirectory() as home:
            output = Path(home)
            code = collector.collect(plan, output, self.fake_runner(
                plan, {'candidatebuild': 130}, interrupted=['candidatebuild']))
            receipt = json.loads((output / 'receipt.json').read_text())
            self.assertEqual(code, 130)
            self.assertEqual(receipt['commands'][1]['status'], 'not-run')

    def test_malformed_shape_retains_final_receipt(self):
        plan = collector.make_plan(collector.ROOT, ['candidatebuild'])
        fake = self.fake_runner(plan)

        def run(argv, cwd, env, output, name, timeout):
            record = fake(argv, cwd, env, output, name, timeout)
            if name == 'candidatebuild':
                (output / record['stdout']).write_text('{"Test":[],"Action":"pass"}\n')
            return record

        with tempfile.TemporaryDirectory() as home:
            output = Path(home)
            self.assertEqual(collector.collect(plan, output, run), 1)
            receipt = json.loads((output / 'receipt.json').read_text())
            self.assertIn('finishedAt', receipt)
            self.assertFalse(receipt['complete'])
            self.assertEqual(len(receipt['commands'][0]['evidence']['parseErrors']), 1)

    def test_fake_sigterm_cleans_group_reaps_and_restores_handler(self):
        handlers = {collector.signal.SIGTERM: 'original-handler'}

        def change_handler(signum, handler):
            previous = handlers[signum]
            handlers[signum] = handler
            return previous

        process = mock.Mock(pid=1234, returncode=-9)

        def wait(timeout=None):
            if timeout is not None:
                handlers[collector.signal.SIGTERM](collector.signal.SIGTERM, None)

        process.wait.side_effect = wait
        with tempfile.TemporaryDirectory() as home, \
                mock.patch.object(collector.signal, 'signal', side_effect=change_handler), \
                mock.patch.object(collector.subprocess, 'Popen', return_value=process), \
                mock.patch.object(collector, 'kill_group') as kill:
            with collector.termination_as_interrupt():
                result = collector.run_command(['go'], Path(home), {}, Path(home), 'fake', 5)
                self.assertTrue(result['interrupted'])
                self.assertEqual(result['effectiveExitCode'], 130)
                self.assertEqual(result['exitCode'], -9)
            self.assertEqual(handlers[collector.signal.SIGTERM], 'original-handler')
            kill.assert_called_once_with(process)
            self.assertEqual(process.wait.call_count, 2)

    def test_fake_probe_failure_remains_failure(self):
        plan = collector.make_plan(collector.ROOT, ['candidatebuild'])
        with tempfile.TemporaryDirectory() as home:
            output = Path(home)
            self.assertEqual(collector.collect(
                plan, output, self.fake_runner(plan, {'go-version': 127})), 127)
            receipt = json.loads((output / 'receipt.json').read_text())
            self.assertFalse(receipt['complete'])
            self.assertTrue(receipt['commands'][0]['evidence']['complete'])

    def test_fake_timeout_kills_owned_group_and_preserves_signal_code(self):
        process = mock.Mock(pid=1234, returncode=-9)
        process.wait.side_effect = [subprocess.TimeoutExpired(['go'], 5), None]
        with tempfile.TemporaryDirectory() as home, \
                mock.patch.object(collector.subprocess, 'Popen', return_value=process) as popen, \
                mock.patch.object(collector, 'kill_group') as kill:
            result = collector.run_command(['go', 'test'], Path(home), {}, Path(home), 'fake', 5)
            self.assertEqual(result['exitCode'], -9)
            self.assertEqual(result['effectiveExitCode'], 124)
            self.assertTrue(result['timedOut'])
            self.assertTrue(popen.call_args.kwargs['start_new_session'])
            kill.assert_called_once_with(process)
            self.assertEqual(process.wait.call_count, 2)

    def test_fake_missing_tool_is_not_success(self):
        with tempfile.TemporaryDirectory() as home, \
                mock.patch.object(collector.subprocess, 'Popen', side_effect=FileNotFoundError('go')):
            result = collector.run_command(['go'], Path(home), {}, Path(home), 'fake', 5)
            self.assertEqual(result['effectiveExitCode'], 127)
            self.assertIsNone(result['exitCode'])
            self.assertIn('launchError', result)


if __name__ == '__main__':
    unittest.main()
