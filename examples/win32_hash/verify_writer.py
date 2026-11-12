"""Build and verify the production writer transport and prepared publication/cleanup."""
import json
import os
from pathlib import Path
import subprocess
import tempfile


def verify():
    source = Path(__file__).parent.resolve()/'testing/writer_probe.c'
    vswhere = Path(os.environ['ProgramFiles(x86)'])/'Microsoft Visual Studio/Installer/vswhere.exe'
    install = subprocess.check_output([str(vswhere), '-latest', '-products', '*', '-property', 'installationPath'], text=True).strip()
    setup = Path(install)/'VC/Auxiliary/Build/vcvars64.bat'
    with tempfile.TemporaryDirectory(prefix='fairy-writer-probe-') as directory:
        root = Path(directory)
        batch = root/'build.cmd'
        batch.write_text(f'@echo off\ncall "{setup}" >nul\nif errorlevel 1 exit /b 1\n'
                         f'cl /nologo /std:c11 /W4 /WX /O2 /MT "{source}" /Fe:probe.exe /Fo:probe.obj user32.lib\n', encoding='utf8')
        built = subprocess.run(['cmd', '/d', '/c', str(batch)], cwd=root, capture_output=True, timeout=60)
        if built.returncode:
            raise RuntimeError(f'native writer build failed: {built.stdout!r} {built.stderr!r}')
        fixture = root/'입력.bin'
        fixture.write_bytes(b'old')
        result = subprocess.run([str(root/'probe.exe'), str(fixture)], capture_output=True, timeout=15)
        if result.returncode or result.stderr:
            raise RuntimeError(f'native writer check failed: {result.stdout!r} {result.stderr!r}')
        return json.loads(result.stdout)


if __name__ == '__main__':
    try:
        print(json.dumps(verify()))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
