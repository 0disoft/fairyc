"""Verify embedded CLI creation with empty PATH, then build/package without GUI."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

def command(args,cwd,code=0,env=None):
    result=subprocess.run(list(map(str,args)),cwd=cwd,capture_output=True,timeout=180,env=env)
    if result.returncode!=code or (code==0 and result.stderr):
        raise RuntimeError(f'command failed: {result.returncode} {result.stdout!r} {result.stderr!r}')
    if code!=0 and result.stdout: raise RuntimeError('rejected creation wrote successful output')
    return result.stdout

def snapshot(root):
    return {p.relative_to(root).as_posix():hashlib.sha256(p.read_bytes()).hexdigest()
            for p in root.rglob('*') if p.is_file()}

def verify(compiler):
    with tempfile.TemporaryDirectory(prefix='fairyc-win32-cli-한글-') as directory:
        parent=Path(directory); binaries=parent/'compiler-only'; binaries.mkdir()
        binary=binaries/'fairyc.exe'; shutil.copy2(compiler,binary)
        isolated=dict(os.environ,PATH='')
        name='cli-tool'; uid='example.cli-tool'
        create=[binary,'new',name,'--uid',uid,'--template','win32']
        output=command(create,parent,env=isolated)
        if output!=b'created Fairy C package "cli-tool" at cli-tool\n': raise RuntimeError('unexpected creation output')
        project=parent/name
        if (project/'fairy.lock').exists(): raise RuntimeError('generation fabricated a lock')
        if len(list(binaries.iterdir()))!=1 or not (project/'win32_app/portable.py').is_file(): raise RuntimeError('compiler is not standalone')
        if any((project/'src'/path).exists() for path in ('hash_file.fy','sha256.fy')): raise RuntimeError('hash domain leaked into starter')
        original=snapshot(project);rejected=0
        command(create,parent,7,isolated);rejected+=1
        if snapshot(project)!=original: raise RuntimeError('duplicate creation modified project')
        for args in (
            ['new','bad','--uid','example.bad','--template','unknown'],
            ['new','bad','--template','win32'],
            ['new','bad','--uid','example.bad','--template'],
            ['new','bad','--uid','example.bad','--uid','example.other'],
            ['new','bad','--uid','example.bad','--template',''],
            ['new','../escape','--uid','example.bad','--template','win32'],
            ['new','con','--uid','example.bad','--template','win32'],
            ['new','bad','--uid','invalid','--template','win32'],
            ['new','a'*128,'--uid','example.bad','--template','win32'],
        ):
            command([binary,*args],parent,2,isolated);rejected+=1
        if {p.name for p in parent.iterdir()}!={'compiler-only',name}: raise RuntimeError('invalid creation left a directory')
        if b'--template <console|win32>' not in command([binary,'new','--help'],parent,env=isolated): raise RuntimeError('new help missing template option')
        # Branding is edited after creation; native output name stays manifest-owned.
        settings=project/'app.toml'
        settings.write_text(settings.read_text(encoding='utf-8').replace('product_name = "cli-tool"','product_name = "CLI 검증 도구 Ω"')
                            .replace('executable_name = "cli-tool.exe"','executable_name = "CliTool.exe"'),encoding='utf-8',newline='\n')
        command([binary,'check'],project)
        command([binary,'check','--locked'],project)
        command([binary,'test','--locked'],project)
        packaged=json.loads(command([sys.executable,'-B','package_release.py','--compiler',binary],project))
        if not packaged['ok'] or packaged['cliCases']!=14 or packaged['asyncCases']!=9 or packaged['files'][0]!='CliTool.exe' or packaged['resources']['windowTitle']!='CLI 검증 도구 Ω':
            raise RuntimeError('generated build/package/branding incomplete')
        if packaged['resources']['strings']['CompanyName']!='' or packaged['version']!='0.1.0': raise RuntimeError('company/version changed')
        # Both forms of console creation remain byte-identical, through the copied EXE.
        first=parent/'default';second=parent/'explicit';first.mkdir();second.mkdir()
        command([binary,'new','basic','--uid','example.basic'],first,env=isolated)
        command([binary,'new','basic','--template','console','--uid','example.basic'],second,env=isolated)
        if snapshot(first/'basic')!=snapshot(second/'basic'): raise RuntimeError('console starter compatibility changed')
        command([binary,'check'],first/'basic');command([binary,'test','--locked'],first/'basic')
        return dict(ok=True,verification='embedded-win32-cli',standaloneCompiler=True,creationWithEmptyPath=True,
            rejectedCreations=rejected,consoleBytesPreserved=True,lockedCheckAndPolicyTest=True,cliCases=packaged['cliCases'],asyncCases=packaged['asyncCases'],
            exeBytes=packaged['exeBytes'],zipBytes=packaged['zipBytes'],zipSha256=packaged['zipSha256'],resources=packaged['resources'],
            guiLaunched=False,inputInjected=False,published=False)

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--compiler',type=Path,required=True)
    args=parser.parse_args()
    try: print(json.dumps(verify(args.compiler.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error))));raise SystemExit(1)
