"""Build isolated mutation gates; verify real file edits through native GUI/CLI."""
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
from source_change_probe import verify_cli


def verify(compiler):
    example = Path(__file__).resolve().parent
    with tempfile.TemporaryDirectory(prefix='fairy-hash-change-') as directory:
        root = Path(directory)
        for name in ('src', 'native'): shutil.copytree(example/name, root/name)
        for name in ('fairy.toml', 'fairy.windows.json'): shutil.copy2(example/name, root/name)
        reader = root/'native/reader.c'; source = reader.read_text(encoding='utf8')
        replacements = (
            ('GENERIC_READ, FILE_SHARE_READ, NULL', 'GENERIC_READ, change_share(value->path), NULL', 2),
            ('ReadFile(value->file,', 'change_read(value->file,', 1),
            ('if (ended) {', 'if (ended) {\n            if (!change_gate(value->path, L"eof")) { goto failed; }', 1),
            ('if (refresh && result.phase != 4) {', 'if (refresh && result.phase != 4) {\n        if (refresh == 1 && !change_gate(value->path, L"cli")) { result.phase = 4; result.error = GetLastError(); memcpy(output, &result, sizeof(result)); return; }', 1),
        )
        for before, after, count in replacements:
            if source.count(before) != count: raise RuntimeError('source-change hook no longer matches transport: '+before)
            source = source.replace(before, after)
        reader.write_text((example/'testing/changing_reader.c').read_text(encoding='utf8')+'\n'+source, encoding='utf8')
        checked([compiler, 'check'], root); checked([compiler, 'build', '--locked', '--release'], root)
        image = root/'build/win32-hash.exe'
        prefix = '\\\\.\\pipe\\fairy-change-'+uuid.uuid4().hex
        env = dict(os.environ, FAIRY_HASH_CHANGE_PIPE=prefix)
        result = subprocess.run([sys.executable, '-B', str(example/'verify_gui.py'), '--exe', str(image), '--source-changes'],
                                capture_output=True, timeout=120, env=env)
        if result.returncode or result.stderr: raise RuntimeError(f'source-change GUI failed: {result.stdout!r} {result.stderr!r}')
        report = json.loads(result.stdout)
        if not report.get('ok') or report.get('verification') != 'actual-win32-hash-source-changes':
            raise RuntimeError('missing GUI source-change report')
        # The CLI process receives its own explicitly scoped gate environment.
        original = os.environ.get('FAIRY_HASH_CHANGE_PIPE')
        try:
            os.environ['FAIRY_HASH_CHANGE_PIPE'] = prefix
            report['cli'] = verify_cli(image, root, prefix)
        finally:
            if original is None: os.environ.pop('FAIRY_HASH_CHANGE_PIPE', None)
            else: os.environ['FAIRY_HASH_CHANGE_PIPE'] = original
        report['cases'] += 1; report['testOnlyGateBuild'] = True
        report['relaxedSharingIsTestOnly'] = True
        return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    args = parser.parse_args()
    try: print(json.dumps(verify(args.compiler.resolve())))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error)))); raise SystemExit(1)
