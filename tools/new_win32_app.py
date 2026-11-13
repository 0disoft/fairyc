"""Create a standalone Fairy C Win32 starter; never overwrite an existing path."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import tempfile
from win32_app.settings import load
from win32_app.resources import prepare
from sync_win32_jobs import install as install_jobs
ROOT=Path(__file__).resolve().parents[1]

def create(destination,name,uid,title,company='',icon=None):
    if not re.fullmatch(r'[a-z][a-z0-9]*(?:-[a-z0-9]+)*',name) or name.upper() in {'CON','PRN','AUX','NUL',*(f'COM{i}' for i in range(1,10)),*(f'LPT{i}' for i in range(1,10))}:
        raise ValueError('name must be a portable lowercase kebab-case package name')
    segment=r'[a-z][a-z0-9]*(?:-[a-z0-9]+)*'
    if not re.fullmatch(segment+r'(?:\.'+segment+r')+',uid):
        raise ValueError('uid must contain lowercase dot-separated segments')
    requested=Path(destination).absolute()
    if requested.name in ('','.','..') or not requested.parent.is_dir(): raise ValueError('destination needs an existing parent and a new directory name')
    target=requested.parent.resolve()/requested.name
    if os.path.lexists(target): raise ValueError('destination already exists; refusing to overwrite')
    with tempfile.TemporaryDirectory(prefix='.fairy-new-',dir=target.parent) as directory:
        stage=Path(directory)/'project'; shutil.copytree(ROOT/'templates/win32_desktop',stage,ignore=shutil.ignore_patterns('__pycache__'))
        install_jobs(stage)
        shutil.copytree(ROOT/'tools/win32_app',stage/'win32_app',ignore=shutil.ignore_patterns('__pycache__'))
        for filename in ('ime.c','ime.h'): shutil.copy2(ROOT/'examples/win32_minimal/native'/filename,stage/'native'/filename)
        shutil.copy2(Path(icon).resolve() if icon else ROOT/'examples/win32_hash/native/app.ico',stage/'native/app.ico')
        shutil.copy2(ROOT/'LICENSE',stage/'LICENSE')
        manifest=f'schema = "fairyc.manifest/v1"\n[package]\nuid = {json.dumps(uid)}\nname = {json.dumps(name)}\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n'
        (stage/'fairy.toml').write_text(manifest,encoding='utf-8',newline='\n')
        values=dict(product_name=title,company_name=company,description=title+' desktop app',
            executable_name=name+'.exe',package_name=name,assembly_name=uid,icon='native/app.ico')
        (stage/'app.toml').write_text('[app]\n'+''.join(f'{key} = {json.dumps(value,ensure_ascii=False)}\n' for key,value in values.items()),encoding='utf-8',newline='\n')
        load(stage); prepare(stage)
        guide=stage/'README.md'; guide.write_text(guide.read_text(encoding='utf-8').replace('@PRODUCT@',title).replace('@PACKAGE@',name),encoding='utf-8',newline='\n')
        # On Windows rename refuses an existing destination, including an empty directory.
        stage.rename(target)
    return dict(ok=True,project=str(target),name=name,uid=uid,guiLaunched=False,inputInjected=False,published=False)

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__); parser.add_argument('destination',type=Path)
    parser.add_argument('--name',required=True); parser.add_argument('--uid',required=True)
    parser.add_argument('--title',default='My Utility'); parser.add_argument('--company',default=''); parser.add_argument('--icon',type=Path)
    args=parser.parse_args()
    try: print(json.dumps(create(args.destination,args.name,args.uid,args.title,args.company,args.icon)))
    except (OSError,ValueError) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
