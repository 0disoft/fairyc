"""Verify the starter domain through CLI without creating a window."""
import argparse
import json
from pathlib import Path
import subprocess
import tomllib
ROOT=Path(__file__).resolve().parent

def verify(image):
    count=0
    for text in ('','hello','한글 Ω','quotes " & spaces'):
        command=subprocess.run([str(image),'--process',text],capture_output=True,timeout=15)
        expected=(('입력: '+text) if text else '내용을 입력하세요.')+'\n'
        if command.returncode or command.stderr or command.stdout!=expected.encode():
            raise RuntimeError('starter CLI output mismatch')
        count+=1
    command=subprocess.run([str(image),'--unknown','text'],capture_output=True,timeout=15)
    if command.returncode!=1 or command.stdout or command.stderr: raise RuntimeError('unknown mode accepted')
    return dict(ok=True,cliCases=count+1,guiLaunched=False,inputInjected=False)

if __name__=='__main__':
    name=tomllib.loads((ROOT/'fairy.toml').read_text(encoding='utf-8'))['package']['name']
    parser=argparse.ArgumentParser(description=__doc__); parser.add_argument('--exe',type=Path,default=ROOT/'build'/f'{name}.exe')
    try: print(json.dumps(verify(parser.parse_args().exe.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
