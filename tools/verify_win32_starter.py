"""Generate outside the repo, build/package, and inspect CLI/resources only."""
import argparse
import json
from pathlib import Path
import subprocess
import sys
import tempfile
ROOT=Path(__file__).resolve().parents[1]

def command(args,cwd):
    result=subprocess.run(list(map(str,args)),cwd=cwd,capture_output=True,timeout=180)
    if result.returncode or result.stderr: raise RuntimeError(f'command failed: {result.stdout!r} {result.stderr!r}')
    return result.stdout

def verify(compiler):
    with tempfile.TemporaryDirectory(prefix='fairy-starter-한글-') as directory:
        parent=Path(directory); project=parent/'독립 도구'; generator=ROOT/'tools/new_win32_app.py'
        create=[sys.executable,'-B',generator,project,'--name','small-tool','--uid','example.small-tool','--title','검증 "도구" Ω']
        reply=json.loads(command(create,parent))
        if not reply['ok'] or not (project/'win32_app/portable.py').is_file(): raise RuntimeError('standalone tooling not bundled')
        if any((project/'src'/name).exists() for name in ('hash_file.fy','sha256.fy','comparison.fy')): raise RuntimeError('starter contains hash product logic')
        command([compiler,'check'],project)
        command([compiler,'test','--locked'],project)
        packaged=json.loads(command([sys.executable,'-B','package_release.py','--compiler',compiler],project))
        if not packaged['ok'] or packaged['cliCases']!=26 or packaged['asyncCases']!=9 or packaged['saveCases']!=12 or packaged['resources']['windowTitle']!='검증 "도구" Ω':
            raise RuntimeError('standalone CLI/package/title verification incomplete')
        if packaged['resources']['strings']['CompanyName']!='': raise RuntimeError('starter invented a publisher')
        original={str(p.relative_to(project)):p.read_bytes() for p in project.rglob('*') if p.is_file()}
        rejected=0
        for args in (create,[*create[:-2],'--title','overwrite']):
            failed=subprocess.run(list(map(str,args)),cwd=parent,capture_output=True,timeout=30)
            if failed.returncode!=1 or json.loads(failed.stdout)['ok']: raise RuntimeError('existing destination overwritten')
            rejected+=1
        after={str(p.relative_to(project)):p.read_bytes() for p in project.rglob('*') if p.is_file()}
        if after!=original: raise RuntimeError('rejected generation modified existing project')
        invalid=parent/'invalid'
        for options in (['--name','CON','--uid','example.tool'],['--name','tool','--uid','example.bad-'],
                        ['--name','tool','--uid','example.tool','--title','bad\nvalue'],
                        ['--name','tool','--uid','example.tool','--icon',str(parent/'missing.ico')]):
            failed=subprocess.run([sys.executable,'-B',str(generator),str(invalid),*options],cwd=parent,capture_output=True,timeout=30)
            if failed.returncode!=1 or json.loads(failed.stdout)['ok'] or invalid.exists(): raise RuntimeError('invalid generation published a project')
            rejected+=1
        return dict(ok=True,verification='standalone-win32-starter',cliCases=26,asyncCases=9,saveCases=12,rejectedCreations=rejected,
            purePolicyTestPassed=True,independentBuild=True,existingProjectPreserved=True,
            exeBytes=packaged['exeBytes'],zipBytes=packaged['zipBytes'],resources=packaged['resources'],
            guiLaunched=False,inputInjected=False,published=False)

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__); parser.add_argument('--compiler',type=Path,required=True)
    args=parser.parse_args()
    try: print(json.dumps(verify(args.compiler.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
