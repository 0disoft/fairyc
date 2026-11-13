"""Build and verify a local unsigned portable ZIP; do not launch the GUI."""
import argparse
import json
from pathlib import Path
import subprocess
import tomllib
import zipfile
from win32_app.resources import prepare
from win32_app.portable import package
from verify_cli import verify
ROOT=Path(__file__).resolve().parent

def checks(image):
    result=verify(image)
    return dict(cliCases=result['cliCases'],asyncCases=result['asyncCases'])

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    source=parser.add_mutually_exclusive_group(required=True)
    source.add_argument('--compiler',type=Path); source.add_argument('--exe',type=Path)
    parser.add_argument('--output',type=Path,default=ROOT/'dist'); args=parser.parse_args()
    try:
        if args.compiler:
            prepare(ROOT)
            built=subprocess.run([str(args.compiler.resolve()),'build','--locked','--release'],cwd=ROOT,capture_output=True,timeout=180)
            if built.returncode: raise RuntimeError(f'build failed: {built.stdout!r} {built.stderr!r}')
            name=tomllib.loads((ROOT/'fairy.toml').read_text(encoding='utf-8'))['package']['name']
            image=ROOT/'build'/f'{name}.exe'
        else: image=args.exe.resolve()
        print(json.dumps(package(image,args.output.resolve(),ROOT,checks)))
    except (OSError,ValueError,RuntimeError,zipfile.BadZipFile,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
