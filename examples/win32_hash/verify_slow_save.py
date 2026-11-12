"""Build test-only native save gates, then verify the actual Fairy GUI from a CLI."""
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


def verify(compiler, gate_exe=None):
    example = Path(__file__).resolve().parent
    with tempfile.TemporaryDirectory(prefix='fairy-hash-slow-save-') as directory:
        root = Path(directory)
        for name in ('src', 'native'):
            shutil.copytree(example/name, root/name)
        for name in ('fairy.toml', 'fairy.windows.json'):
            shutil.copy2(example/name, root/name)
        writer = root/'native/writer.c'
        source = writer.read_text(encoding='utf8')
        replacements = (
            ('static BOOL writer_check(writer *value) {', 'static BOOL writer_check(writer *value) {\n    if (!delayed_save_gate(value->target, L"check")) { return FALSE; }'),
            ('static BOOL writer_create(writer *value) {', 'static BOOL writer_create(writer *value) {\n    if (!delayed_save_gate(value->target, L"create")) { return FALSE; }'),
            ('static BOOL writer_write(writer *value) {', 'static BOOL writer_write(writer *value) {\n    if (!delayed_save_gate(value->target, L"write")) { return FALSE; }'),
            ('ok = FlushFileBuffers(value->file);', 'ok = delayed_save_gate(value->target, L"flush") && FlushFileBuffers(value->file);'),
            ('ok = SetFileInformationByHandle(value->file, FileRenameInfo, rename, capacity);', 'ok = delayed_save_gate(value->target, L"publish") && SetFileInformationByHandle(value->file, FileRenameInfo, rename, capacity);'),
            ('if (!CloseHandle(value->file))', 'if (!(delayed_save_gate(value->target, L"cleanup") && CloseHandle(value->file)))'),
        )
        for before, after in replacements:
            if source.count(before) != 1:
                raise RuntimeError('slow-save test hook no longer matches production transport: '+before)
            source = source.replace(before, after)
        writer.write_text((example/'testing/delayed_writer.c').read_text(encoding='utf8')+'\n'+source, encoding='utf8')
        checked([compiler, 'check'], root)
        checked([compiler, 'build', '--locked', '--release'], root)
        if gate_exe:
            gate_exe.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(root/'build/win32-hash.exe', gate_exe)
        reports = []
        for mode in ('main', 'close-write', 'close-publish', 'close-cleanup'):
            env = dict(os.environ, FAIRY_HASH_SAVE_TEST_PIPE='\\\\.\\pipe\\fairy-save-'+uuid.uuid4().hex)
            result = subprocess.run([sys.executable, '-B', str(example/'verify_gui.py'), '--exe', str(root/'build/win32-hash.exe'), '--slow-save', mode],
                                    capture_output=True, timeout=90, env=env)
            if result.returncode or result.stderr:
                raise RuntimeError(f'pending-save GUI verification failed ({mode}): {result.stdout!r} {result.stderr!r}')
            report = json.loads(result.stdout)
            if not report.get('ok') or report.get('verification') != 'actual-win32-hash-slow-save':
                raise RuntimeError('missing pending-save GUI report')
            reports.append(report)
        return dict(ok=True, verification='actual-win32-hash-slow-save', testOnlyGateBuild=True,
                    cases=sum(report['cases'] for report in reports), profiles=reports)



if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--gate-exe', type=Path, help='retain the test-only image for focused CLI reruns')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.compiler.resolve(), args.gate_exe.resolve() if args.gate_exe else None)))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
