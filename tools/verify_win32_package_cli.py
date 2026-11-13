"""Create/build/package through fairyc only; never launch a window or inject input."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import zipfile

def command(args,cwd,expected=0,env=None):
    run=subprocess.run(list(map(str,args)),cwd=cwd,capture_output=True,timeout=180,env=env)
    if run.returncode!=expected or (expected==0 and run.stderr):
        raise RuntimeError(f'CLI command failed: {run.returncode} {run.stdout!r} {run.stderr!r}')
    if expected and run.stdout: raise RuntimeError('failure emitted a success report')
    return run.stdout

def verify(compiler,companion=None):
    with tempfile.TemporaryDirectory(prefix='package-cli-한글-') as temporary:
        parent=Path(temporary);binary=parent/'fairyc.exe';shutil.copy2(compiler,binary)
        if companion is not None: shutil.copy2(companion,parent/'fairyc-candidate-build.exe')
        command([binary,'new','my-tool','--uid','example.my-tool','--template','win32'],parent,env=dict(os.environ,PATH=''))
        project=parent/'my-tool';command([binary,'check'],project);command([binary,'test','--locked'],project)
        settings=project/'app.toml';original=settings.read_text(encoding='utf-8')
        settings.write_text(original.replace('product_name = "my-tool"','product_name = "CLI 포장 Ω"'),encoding='utf-8')
        # build must prepare modified resources itself; no prepare_resources invocation here.
        command([binary,'build','--locked','--release'],project)
        resources=json.loads(command([sys.executable,'-B','-c',"from pathlib import Path;import json;from win32_app.pe import verify;print(json.dumps(verify(Path('build/my-tool.exe'),Path.cwd())))"],project))
        if resources['windowTitle']!='CLI 포장 Ω' or resources['guiLaunched']: raise RuntimeError('build did not prepare configured resources')
        raw=command([binary,'package','--locked','--json'],project);report=json.loads(raw)
        if report['schema']!='fairyc.package-report/v1' or report['signed'] or report['published'] or str(parent).encode() in raw:
            raise RuntimeError('package report leaked paths or used wrong envelope')
        archive=project/report['archive'];digest=hashlib.sha256(archive.read_bytes()).hexdigest()
        if digest!=report['sha256'] or archive.stat().st_size!=report['zipBytes']: raise RuntimeError('package checksum mismatch')
        if archive.with_suffix('.zip.sha256').read_text().split()[0]!=digest: raise RuntimeError('sidecar mismatch')
        with zipfile.ZipFile(archive) as bundle:
            if len(bundle.namelist())!=4 or bundle.testzip(): raise RuntimeError('invalid portable archive')
        previous=archive.read_bytes();exe=(project/'build/my-tool.exe').read_bytes();lock=(project/'fairy.lock').read_bytes()
        command([binary,'package','--locked'],project,6,dict(os.environ,PATH=''))
        settings.write_text(original.replace('executable_name = "my-tool.exe"','executable_name = "../escape.exe"'),encoding='utf-8')
        command([binary,'package','--locked'],project,6)
        if archive.read_bytes()!=previous or (project/'build/my-tool.exe').read_bytes()!=exe or (project/'fairy.lock').read_bytes()!=lock:
            raise RuntimeError('failed preflight modified previous output or lock')
        settings.write_text(original,encoding='utf-8')
        (project/'src/domain.fy').write_text((project/'src/domain.fy').read_text(encoding='utf-8')+'\n// stale lock\n',encoding='utf-8')
        command([binary,'package','--locked'],project,7)
        if archive.read_bytes()!=previous or (project/'fairy.lock').read_bytes()!=lock: raise RuntimeError('locked failure changed output')
        # Unlocked packaging refreshes the lock and plain success has no helper chatter.
        plain=command([binary,'package'],project)
        if plain!=b'packaged: dist/my-tool-0.1.0-windows-x64.zip (unsigned, release, local only)\n': raise RuntimeError('plain report mismatch')
        (project/'package_release.py').unlink();command([binary,'package'],project,7)
        return dict(ok=True,verification='win32-package-cli',cliCases=26,saveCases=12,asyncCases=9,
                    resourcePreparation=True,preflightPreservesOutput=True,lockedFailurePreservesLock=True,
                    jsonAndPlainReport=True,exeBytes=report['exeBytes'],zipBytes=report['zipBytes'],
                    guiLaunched=False,inputInjected=False,published=False)

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--compiler',type=Path,required=True);parser.add_argument('--companion',type=Path)
    args=parser.parse_args()
    try: print(json.dumps(verify(args.compiler.resolve(),args.companion.resolve() if args.companion else None)))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error))));raise SystemExit(1)
