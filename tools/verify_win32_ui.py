"""Compile real common UI code, check conversion/DPI/argument policy, never create a window."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
ROOT=Path(__file__).resolve().parents[1]
def verify():
    locator=Path(os.environ['ProgramFiles(x86)'])/'Microsoft Visual Studio/Installer/vswhere.exe'
    installation=subprocess.check_output([str(locator),'-latest','-products','*','-property','installationPath'],text=True).strip()
    setup=Path(installation)/'VC/Auxiliary/Build/vcvars64.bat'
    with tempfile.TemporaryDirectory(prefix='fairy-ui-core-') as directory:
        root=Path(directory);batch=root/'build.cmd';source=ROOT/'modules/win32_ui'
        batch.write_text(f'@echo off\ncall "{setup}" >nul\nif errorlevel 1 exit /b 1\n'
            f'cl /nologo /std:c11 /W4 /WX /O2 /MT /I"{source}" "{ROOT / "tools/testing/win32_ui_probe.c"}" "{source / "native/ui_core.c"}" /Fe:probe.exe user32.lib gdi32.lib ole32.lib\n',encoding='utf-8')
        built=subprocess.run(['cmd','/d','/c',str(batch)],cwd=root,capture_output=True,timeout=60)
        if built.returncode: raise RuntimeError(f'UI core compilation failed: {built.stdout!r} {built.stderr!r}')
        result=subprocess.run([str(root/'probe.exe')],cwd=root,capture_output=True,timeout=15)
        if result.returncode or result.stderr: raise RuntimeError(f'UI core check failed: {result.stdout!r} {result.stderr!r}')
        return json.loads(result.stdout)
if __name__=='__main__':
    try: print(json.dumps(verify()))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error))));raise SystemExit(1)
