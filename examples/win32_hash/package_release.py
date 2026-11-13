"""Build or select a release EXE, verify its portable ZIP, and write a SHA-256 sidecar."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile
import zipfile
from prepare_resources import ROOT,metadata,prepare
from verify_resources import verify as verify_resources
from verify_package import payload,verify as verify_package


def package(image, output):
    version,_,_ = metadata()
    pe = verify_resources(image)
    if pe['certificateTablePresent']:
        raise RuntimeError('this unsigned-package template does not describe signed images; update distribution instructions first')
    files = payload(image)
    output.mkdir(parents=True,exist_ok=True)
    archive = output/f'fairy-hash-{version}-windows-x64.zip'
    with tempfile.TemporaryDirectory(prefix='.fairy-package-',dir=output) as directory:
        pending = Path(directory)/archive.name
        with zipfile.ZipFile(pending,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as bundle:
            for name,data in files.items():
                entry = zipfile.ZipInfo(name,date_time=(1980,1,1,0,0,0))
                entry.create_system = 3
                entry.external_attr = (0o100755 if name.endswith('.exe') else 0o100644)<<16
                entry.compress_type = zipfile.ZIP_DEFLATED
                bundle.writestr(entry,data,compresslevel=9)
        result = verify_package(pending)
        checksum = archive.with_suffix(archive.suffix+'.sha256')
        sidecar = Path(directory)/checksum.name
        sidecar.write_text(f'{result["zipSha256"]}  {archive.name}\n',encoding='ascii',newline='\n')
        os.replace(pending,archive)
        os.replace(sidecar,checksum)
    result.update(archive=str(archive),checksumFile=str(checksum),published=False,signed=False)
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
