"""Refresh the deterministic compiler-embedded starter from canonical repo sources."""
import argparse
import hashlib
import json
from pathlib import Path
import tempfile
import zipfile
from new_win32_app import create, ROOT

NAME='fairyc-template-name'
UID='example.fairyc-template-uid'
TITLE='FAIRYC_TEMPLATE_TITLE'
OUTPUT=ROOT/'internal/projectscaffold/assets'

def normalized(path):
    data=path.read_bytes()
    return data if path.suffix=='.ico' else data.replace(b'\r\n',b'\n')

def bundle():
    inputs=[ROOT/'tools/bundle_win32_starter.py',ROOT/'tools/new_win32_app.py',ROOT/'LICENSE',
            ROOT/'examples/win32_hash/native/app.ico',ROOT/'examples/win32_minimal/native/ime.c',
            ROOT/'examples/win32_minimal/native/ime.h']
    inputs.extend(p for p in (ROOT/'templates/win32_desktop').rglob('*') if p.is_file() and '__pycache__' not in p.parts)
    inputs.extend(p for p in (ROOT/'tools/win32_app').rglob('*') if p.is_file() and '__pycache__' not in p.parts)
    with tempfile.TemporaryDirectory(prefix='fairy-starter-bundle-') as directory:
        project=Path(directory)/'project'; create(project,NAME,UID,TITLE)
        OUTPUT.mkdir(parents=True,exist_ok=True)
        archive=OUTPUT/'win32.zip'
        with zipfile.ZipFile(archive,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as package:
            for path in sorted(project.rglob('*')):
                if not path.is_file(): continue
                name=path.relative_to(project).as_posix(); data=normalized(path)
                if name in ('app.toml','fairy.toml','native/app.rc','README.md'):
                    for before,after in ((NAME,'@@NAME@@'),(UID,'@@UID@@'),(TITLE,'@@TITLE@@')):
                        data=data.replace(before.encode(),after.encode())
                entry=zipfile.ZipInfo(name,date_time=(1980,1,1,0,0,0)); entry.create_system=3
                entry.external_attr=0o100644<<16; entry.compress_type=zipfile.ZIP_DEFLATED
                package.writestr(entry,data,compresslevel=9)
    receipt=dict(version='0.1.0',language='fairyc-language/20',bundleSha256=hashlib.sha256(archive.read_bytes()).hexdigest(),
        inputs={p.relative_to(ROOT).as_posix():hashlib.sha256(normalized(p)).hexdigest() for p in sorted(set(inputs))})
    (OUTPUT/'win32.inputs.json').write_text(json.dumps(receipt,indent=2,ensure_ascii=False)+'\n',encoding='utf-8',newline='\n')
    return dict(ok=True,archive=str(archive),bytes=archive.stat().st_size,sourceInputs=len(receipt['inputs']))

if __name__=='__main__':
    argparse.ArgumentParser(description=__doc__).parse_args()
    print(json.dumps(bundle()))
