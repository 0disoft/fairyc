"""Plan focused existing Linux tests; only --execute on Linux amd64 runs them."""

import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import platform
import re
import signal
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
# Explicit roots, never ./... or a prefix regex. Full CLI construction is opt-in.
GROUPS = {
    'candidatebuild': ('internal/candidatebuild', (
        'TestLinuxPacketBuildAndPublication', 'TestLinuxPacketProcessArguments')),
    'runtime': ('internal/cli', (
        'TestLinuxGNUCompanionRuntime', 'TestLinuxGNUCompanionRuntimeAssets',
        'TestLinuxGNUCompanionNative')),
    'trap-runtime': ('internal/cruntime', ('TestLinuxTrapReceiptProgressNative',)),
    'cli': ('internal/cli', ('TestLinuxFileCLI',)),
    'fetch-native': ('internal/cli', (
        'TestLinuxRemoteObservedReadNative', 'TestLinuxRemoteOfflineNative',
        'TestLinuxRemoteOnlineNative')),
    'fetch-host': ('internal/candidatehost', (
        'TestLinuxPackageDirectoryAdapterNative', 'TestLinuxPackageFileAdapterNative',
        'TestLinuxClockAdapterNative', 'TestLinuxHTTPSWorkerRejectsUntrustedNative',
        'TestLinuxHTTPSParentProcessNative', 'TestLinuxHTTPSParentRejectsUntrustedNative',
        'TestLinuxHTTPSParentTLSDeadlineNative')),
}
DEFAULT_GROUPS = ('candidatebuild', 'trap-runtime')
ENVIRONMENT = {
    'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOSUMDB': 'off',
    'GOWORK': 'off', 'GOENV': 'off', 'GOFLAGS': '',
    'GOOS': 'linux', 'GOARCH': 'amd64', 'CGO_ENABLED': '1',
}
PROBES = (
    ('git-sha', ['git', '--no-optional-locks', 'rev-parse', 'HEAD']),
    ('git-dirty', ['git', '--no-optional-locks', 'status', '--porcelain',
                   '--untracked-files=normal']),
    ('go-version', ['go', 'version']),
    ('go-env', ['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'CGO_ENABLED']),
    ('gcc-version', ['gcc', '--version']),
    ('clang-version', ['clang', '--version']),
    ('libcurl-version', ['pkg-config', '--modversion', 'libcurl']),
)


def utc_now():
    return datetime.now(timezone.utc).isoformat()


def make_plan(root, groups=None, timeout=900):
    selected = list(dict.fromkeys(DEFAULT_GROUPS if groups is None else groups))
    if not selected or any(group not in GROUPS for group in selected):
        raise ValueError('select at least one known group')
    commands = []
    for group in selected:
        package, tests = GROUPS[group]
        owners = {}
        for path in sorted((root / package).glob('*_test.go')):
            source = path.read_text(encoding='utf-8')
            for name in tests:
                if re.search(r'^func ' + name + r'\(t \*testing\.T\)', source, re.M):
                    owners.setdefault(name, []).append(path.relative_to(root).as_posix())
        if any(len(owners.get(name, [])) != 1 for name in tests):
            raise ValueError(f'{group}: missing or ambiguous test root; review current source')
        commands.append({
            'group': group, 'package': './' + package, 'roots': list(tests),
            'sources': {name: owners[name][0] for name in tests},
            'argv': ['go', 'test', '-mod=readonly', '-count=1', '-json',
                     '-timeout', f'{timeout}s', './' + package,
                     '-run', '^(' + '|'.join(tests) + ')$'],
            'timeoutSeconds': timeout + 30,
        })
    return {'mode': 'plan', 'cwd': str(root), 'groups': selected,
            'environmentOverrides': ENVIRONMENT, 'commands': commands,
            'requiredOptIns': {},
            'note': 'Plan only. No tools invoked. Full CLI requires --group cli. '
                    'Selected roots require no FAIRYC_* integration opt-ins.'}


def child_environment():
    # Helper modes and unrelated integration opt-ins must not leak into this replay.
    env = {key: value for key, value in os.environ.items()
           if not key.upper().startswith('FAIRYC_')}
    env.update(ENVIRONMENT)
    return env


def kill_group(process):
    # Each real command owns a fresh Linux session, including compiler/test children.
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass


@contextmanager
def termination_as_interrupt():
    def interrupt(signum, frame):
        raise KeyboardInterrupt

    previous = signal.signal(signal.SIGTERM, interrupt)
    try:
        yield
    finally:
        signal.signal(signal.SIGTERM, previous)


def run_command(argv, cwd, env, output, name, timeout):
    stdout = output / (name + '.stdout')
    stderr = output / (name + '.stderr')
    record = {'argv': argv, 'cwd': str(cwd), 'startedAt': utc_now(),
              'stdout': stdout.name, 'stderr': stderr.name,
              'timeoutSeconds': timeout, 'timedOut': False, 'interrupted': False,
              'exitCode': None}
    started = time.monotonic()
    process = None
    with stdout.open('wb') as out, stderr.open('wb') as err:
        try:
            process = subprocess.Popen(argv, cwd=cwd, env=env,
                                       stdin=subprocess.DEVNULL, stdout=out, stderr=err,
                                       start_new_session=True)
            process.wait(timeout=timeout)
        except OSError as error:
            record['launchError'] = str(error)
            record['effectiveExitCode'] = 127
        except subprocess.TimeoutExpired:
            record['timedOut'] = True
        except KeyboardInterrupt:
            record['interrupted'] = True
        finally:
            if process is not None:
                # Also reap descendants left in our session after a successful parent.
                kill_group(process)
                process.wait()
        if process is not None:
            record['exitCode'] = process.returncode
        if 'effectiveExitCode' not in record:
            record['effectiveExitCode'] = (
                130 if record['interrupted'] else 124 if record['timedOut'] else
                128 - process.returncode if process.returncode < 0 else process.returncode)
    record['durationSeconds'] = round(time.monotonic() - started, 6)
    return record


def summarize_tests(path, roots):
    """Keep root skips, missing events and child skips distinct from complete passes."""
    results = {name: {'status': 'missing', 'skips': []} for name in roots}
    errors = []
    package_failed = False
    with path.open(encoding='utf-8', errors='replace') as stream:
        for number, line in enumerate(stream, 1):
            try:
                event = json.loads(line)
                if not isinstance(event, dict):
                    raise ValueError('not an object')
                if (not isinstance(event.get('Action'), str) or
                        any(not isinstance(event.get(field, ''), str)
                            for field in ('Test', 'Package', 'Output'))):
                    raise ValueError('invalid Go event field type')
            except ValueError:
                errors.append(f'invalid Go JSON at line {number}')
                continue
            test, action = event.get('Test', ''), event.get('Action')
            if action in ('fail', 'build-fail'):
                package_failed = True
            root = test.split('/')[0]
            if root not in results:
                continue
            result = results[root]
            if action == 'skip':
                result['skips'].append(test)
            if test == root and action in ('pass', 'fail', 'skip'):
                if result['status'] != 'missing':
                    errors.append(f'duplicate terminal event: {root}')
                result['status'] = action
    for result in results.values():
        if result['status'] == 'pass' and result['skips']:
            result['status'] = 'partial'
    return {'roots': results, 'parseErrors': errors, 'packageFailed': package_failed,
            'complete': not errors and not package_failed and
                        all(r['status'] == 'pass' for r in results.values())}


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')


def collect(plan, output, runner=run_command):
    env = child_environment()
    receipt = {'mode': 'execute', 'startedAt': utc_now(),
               'host': {'system': platform.system(), 'machine': platform.machine(),
                        'python': platform.python_version()},
               'plan': plan, 'tools': [], 'commands': [], 'complete': False}
    receipt_path = output / 'receipt.json'
    write_json(receipt_path, receipt)
    first_failure = 0
    interrupted = False
    try:
        for name, argv in PROBES:
            record = runner(argv, Path(plan['cwd']), env, output, name, 30)
            record['name'] = name
            receipt['tools'].append(record)
            if record['effectiveExitCode'] and not first_failure:
                first_failure = record['effectiveExitCode']
            write_json(receipt_path, receipt)
            if record.get('interrupted'):
                raise KeyboardInterrupt
        for command in plan['commands']:
            record = runner(command['argv'], Path(plan['cwd']), env, output,
                            command['group'], command['timeoutSeconds'])
            record['group'] = command['group']
            receipt['commands'].append(record)
            write_json(receipt_path, receipt)
            evidence = summarize_tests(output / record['stdout'], command['roots'])
            record['evidence'] = evidence
            code = record['effectiveExitCode']
            if not code and (evidence['packageFailed'] or evidence['parseErrors']):
                code = 1
            if code and not first_failure:
                first_failure = code
            write_json(receipt_path, receipt)
            if record.get('interrupted'):
                raise KeyboardInterrupt
    except KeyboardInterrupt:
        interrupted = True
        recorded = {record['group'] for record in receipt['commands']}
        for command in plan['commands']:
            if command['group'] not in recorded:
                receipt['commands'].append({**command, 'status': 'not-run',
                                            'reason': 'collector interrupted'})
    receipt['complete'] = not interrupted and not first_failure and all(
        record.get('evidence', {}).get('complete', False) for record in receipt['commands'])
    receipt['exitCode'] = 130 if interrupted else first_failure or (0 if receipt['complete'] else 3)
    receipt['finishedAt'] = utc_now()
    write_json(receipt_path, receipt)
    return receipt['exitCode']


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True, help='new named output directory')
    parser.add_argument('--group', action='append', choices=GROUPS,
                        help='repeat for bounded selection; replaces the small default set')
    parser.add_argument('--timeout-seconds', type=int, default=900,
                        help='per-group Go deadline, 1..3600; outer cleanup adds 30 seconds')
    parser.add_argument('--execute', action='store_true')
    args = parser.parse_args(argv)
    if not 1 <= args.timeout_seconds <= 3600:
        parser.error('--timeout-seconds must be 1..3600')
    if args.execute and (sys.platform != 'linux' or platform.machine().lower() not in ('x86_64', 'amd64')):
        parser.error('--execute requires native Linux amd64; Windows tests are not a Linux pass')
    try:
        plan = make_plan(ROOT, args.group, args.timeout_seconds)
        output = args.output.resolve()
        output.mkdir(parents=True, exist_ok=False)
    except (OSError, ValueError) as error:
        parser.error(str(error))
    write_json(output / 'plan.json', plan)
    if not args.execute:
        print(f'Plan only (no subprocesses): {output / "plan.json"}')
        return 0
    with termination_as_interrupt():
        return collect(plan, output)


if __name__ == '__main__':
    sys.exit(main())
