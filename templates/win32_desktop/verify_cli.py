"""Verify the starter domain through CLI without creating a window."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import tomllib
from verify_save import verify as verify_saves
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
    count+=1
    with tempfile.TemporaryDirectory(prefix='job-한글-') as directory:
        root=Path(directory);source=root/'입력 Ω.bin';data=(b'abc\n'+bytes(range(256)))*4096
        def scan(payload,after=None):
            nonlocal count
            source.write_bytes(payload)
            args=[str(image),'--scan',str(source)]+(['--cancel-after',str(after)] if after is not None else [])
            command=subprocess.run(args,capture_output=True,timeout=30)
            want=2 if after is not None and after<=len(payload) else 0
            if command.returncode!=want or command.stderr: raise RuntimeError('scan exit mismatch')
            rows=[json.loads(line) for line in command.stdout.splitlines()];result=rows[-1]
            if result['event']!='result' or result['cancelled']!=(want==2) or result['ok']!=(want==0): raise RuntimeError('scan terminal state mismatch')
            if result['done']>len(payload) or result['lines']!=payload[:result['done']].count(b'\n'): raise RuntimeError('scan bytes were not consumed correctly')
            if want==0 and (result['done']!=len(payload) or result['percent']!=100): raise RuntimeError('verified completion missing')
            if want==2 and (result['percent']>=100 or result['errorCode']!=8): raise RuntimeError('cancel reported a valid result')
            if any(row['percent']>=100 for row in rows[:-1]) or any(b['done']<a['done'] or b['elapsedMs']<a['elapsedMs'] for a,b in zip(rows,rows[1:])):
                raise RuntimeError('progress overflow/regression or premature completion')
            if len(payload)>65536 and want==0 and not any(0<row['percent']<100 for row in rows): raise RuntimeError('no intermediate progress observed')
            count+=1
        for payload in (b'',b'abc\n\n',data): scan(payload)
        for after in (0,1,len(data),len(data)+1): scan(data,after)
        command=subprocess.run([str(image),'--scan',str(root/'missing.bin')],capture_output=True,timeout=15)
        if command.returncode!=1 or json.loads(command.stdout.splitlines()[-1])['errorCode']!=2: raise RuntimeError('missing scan source accepted')
        count+=1
        command=subprocess.run([str(image),'--scan',str(source),'--cancel-after','-1'],capture_output=True,timeout=15)
        if command.returncode!=1 or command.stdout: raise RuntimeError('invalid cancel threshold accepted')
        count+=1
    saved=verify_saves(image)
    return dict(ok=True,cliCases=count+saved,asyncCases=9,saveCases=saved,guiLaunched=False,inputInjected=False)

if __name__=='__main__':
    name=tomllib.loads((ROOT/'fairy.toml').read_text(encoding='utf-8'))['package']['name']
    parser=argparse.ArgumentParser(description=__doc__); parser.add_argument('--exe',type=Path,default=ROOT/'build'/f'{name}.exe')
    try: print(json.dumps(verify(parser.parse_args().exe.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
