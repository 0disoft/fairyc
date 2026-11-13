"""Build or select a release EXE, verify its portable ZIP, and write a SHA-256 sidecar."""
import argparse
import json
from pathlib import Path
import subprocess
import zipfile
from prepare_resources import ROOT,prepare
from verify_package import checks
from win32_app.portable import package as portable_package


def package(image, output, root=ROOT):
    result=portable_package(image,output,root,checks)
    result["verification"]="fairy-hash-portable-package"
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument('--compiler',type=Path,help='build --locked --release before packaging')
    source.add_argument('--exe',type=Path,help='package an already-built release with matching resources')
    parser.add_argument('--output',type=Path,default=ROOT/'dist')
    args = parser.parse_args()
    try:
        if args.compiler:
            prepare()
            command = subprocess.run([str(args.compiler.resolve()),'build','--locked','--release'],cwd=ROOT,
                                     capture_output=True,timeout=180)
            if command.returncode: raise RuntimeError('release build failed: '+command.stderr.decode('utf-8',errors='replace'))
            image = ROOT/'build/win32-hash.exe'
        else: image = args.exe.resolve()
        print(json.dumps(package(image,args.output.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired,zipfile.BadZipFile) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
