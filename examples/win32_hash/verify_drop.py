"""Check production Shell drop decoding with HDROP data; never create a window."""
import json
import os
from pathlib import Path
import subprocess
import tempfile


def verify():
    source = Path(__file__).resolve().parent/'testing/drop_probe.c'
    vswhere = Path(os.environ['ProgramFiles(x86)'])/'Microsoft Visual Studio/Installer/vswhere.exe'
    install = subprocess.check_output([str(vswhere),'-latest','-products','*','-property','installationPath'],text=True).strip()
    setup = Path(install)/'VC/Auxiliary/Build/vcvars64.bat'
    with tempfile.TemporaryDirectory(prefix='fairy-drop-') as directory:
        root = Path(directory)
        batch = root/'build.cmd'
        batch.write_text(f'@echo off\ncall "{setup}" >nul\nif errorlevel 1 exit /b 1\n'
                         f'cl /nologo /utf-8 /std:c11 /W4 /WX /O2 /MT "{source}" /Fe:probe.exe /Fo:probe.obj shell32.lib\n',encoding='utf-8')
        built = subprocess.run(['cmd','/d','/c',str(batch)],cwd=root,capture_output=True,timeout=60)
        if built.returncode: raise RuntimeError(f'drop probe build failed: {built.stdout!r} {built.stderr!r}')
        tested = subprocess.run([str(root/'probe.exe')],capture_output=True,timeout=15)
        if tested.returncode or tested.stderr: raise RuntimeError(f'drop decoding failed: {tested.returncode} {tested.stdout!r} {tested.stderr!r}')
        return json.loads(tested.stdout)


if __name__ == '__main__':
    try: print(json.dumps(verify()))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
