"""Build test-only native I/O gates, then verify the actual Fairy GUI from a CLI."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import uuid
from profile_costs import checked


def verify(compiler):
    example = Path(__file__).resolve().parent
    with tempfile.TemporaryDirectory(prefix='fairy-hash-slow-io-') as directory:
        root = Path(directory)
        for name in ('src', 'native'):
            shutil.copytree(example/name, root/name)
        for name in ('fairy.toml', 'fairy.windows.json'):
            shutil.copy2(example/name, root/name)
        reader = root/'native/reader.c'
        source = reader.read_text(encoding='utf8')
        for before, after in (('value->file = CreateFileW(', 'value->file = delayed_open('),
                              ('ReadFile(value->file,', 'delayed_read(value->file,'),
                              ('GetFileInformationByHandleEx(value->file,', 'delayed_metadata(value->file,')):
            if source.count(before) != 1:
                raise RuntimeError('slow-I/O test hook no longer matches production transport')
            source = source.replace(before, after)
        reader.write_text((example/'testing/delayed_reader.c').read_text(encoding='utf8')+'\n'+source, encoding='utf8')
        checked([compiler, 'check'], root)
        checked([compiler, 'build', '--locked', '--release'], root)
        env = dict(os.environ, FAIRY_HASH_TEST_PIPE='\\\\.\\pipe\\fairy-hash-'+uuid.uuid4().hex)
        result = subprocess.run([sys.executable, '-B', str(example/'verify_gui.py'), '--exe', str(root/'build/win32-hash.exe'), '--slow-io'],
                                capture_output=True, timeout=90, env=env)
        if result.returncode or result.stderr:
            raise RuntimeError(f'pending-I/O GUI verification failed: {result.stdout!r} {result.stderr!r}')
        report = json.loads(result.stdout)
        if not report.get('ok') or report.get('verification') != 'actual-win32-hash-slow-io':
            raise RuntimeError('missing pending-I/O GUI report')
        report['testOnlyGateBuild'] = True
        return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.compiler.resolve())))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
