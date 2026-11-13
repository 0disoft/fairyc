"""Validate a portable ZIP, then run only resource inspection and CLI hash cases."""
import argparse
import hashlib
import json
from pathlib import Path
import tempfile
import subprocess
import zipfile
from prepare_resources import ROOT,metadata
from verify_resources import verify as resources
from verify_cli import verify as cli

FILES = ('FairyHash.exe','README.txt','LICENSE.txt')


def payload(image, root=ROOT):
    version,_,_ = metadata(root)
    guide = (root/'distribution/README.ko.txt').read_text(encoding='utf-8').replace('@VERSION@',version)
    files = dict(zip(FILES,(image.read_bytes(),guide.encode('utf-8-sig'),(root.parents[1]/'LICENSE').read_bytes())))
    files['SHA256SUMS.txt'] = ''.join(f'{hashlib.sha256(files[name]).hexdigest()}  {name}\n' for name in FILES).encode('ascii')
    return files


def verify(archive, root=ROOT):
    with zipfile.ZipFile(archive) as package:
        entries = package.infolist()
        if len(entries) != 4 or {entry.filename for entry in entries} != {*FILES,'SHA256SUMS.txt'}:
            raise RuntimeError('unexpected/duplicate files in portable ZIP')
        if sum(entry.file_size for entry in entries) > 32*1024*1024:
            raise RuntimeError('portable ZIP exceeds expected size bound')
        if any(entry.flag_bits & 1 or (entry.external_attr >> 16) & 0o170000 == 0o120000 for entry in entries):
            raise RuntimeError('encrypted/symlink archive entry is not supported')
        if package.testzip() is not None: raise RuntimeError('ZIP CRC mismatch')
        files = {entry.filename:package.read(entry) for entry in entries}
    expected = ''.join(f'{hashlib.sha256(files[name]).hexdigest()}  {name}\n' for name in FILES).encode('ascii')
    if files['SHA256SUMS.txt'] != expected: raise RuntimeError('package SHA-256 checksum mismatch')
    if files['LICENSE.txt'] != (root.parents[1]/'LICENSE').read_bytes(): raise RuntimeError('license text mismatch')
    version,_,_ = metadata(root)
    guide = (root/'distribution/README.ko.txt').read_text(encoding='utf-8').replace('@VERSION@',version).encode('utf-8-sig')
    if files['README.txt'] != guide: raise RuntimeError('packaged instructions/version mismatch')
    with tempfile.TemporaryDirectory(prefix='Fairy Hash 패키지-') as directory:
        target = Path(directory)/'FairyHash.exe'
        target.write_bytes(files['FairyHash.exe'])
        pe,commands = resources(target,root),cli(target)
    return dict(ok=True,verification='fairy-hash-portable-package',version=version,
                files=list(files),zipBytes=archive.stat().st_size,exeBytes=len(files['FairyHash.exe']),
                zipSha256=hashlib.sha256(archive.read_bytes()).hexdigest(),resources=pe,cliCases=commands['cases'],
                temporaryExtractionRemoved=True,guiLaunched=False,inputInjected=False)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('archive',type=Path)
    args = parser.parse_args()
    try: print(json.dumps(verify(args.archive.resolve())))
    except (OSError,ValueError,RuntimeError,zipfile.BadZipFile,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
