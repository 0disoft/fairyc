"""Exercise single-entry checksum import/export through the actual CLI only."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile


def verify(image):
    count=0
    with tempfile.TemporaryDirectory(prefix='fairy-checksum-한글-') as directory:
        root=Path(directory); source=root/'입력 Ω.txt'; source.write_bytes(b'abc')
        actual=hashlib.sha256(b'abc').hexdigest(); checksum=root/'입력.sha256'; report=root/'result.json'
        def run(args, code=0, error=None, text=None):
            nonlocal count
            command=subprocess.run([str(image),*map(str,args)],capture_output=True,timeout=30)
            if command.returncode!=code or command.stderr: raise RuntimeError(f'checksum exit/stderr: {command.returncode} {command.stderr!r}')
            if text is not None:
                if command.stdout!=text: raise RuntimeError('checksum text mismatch')
                value=None
            else:
                value=json.loads(command.stdout)
                if error is not None:
                    if value!=dict(ok=False,errorCode=error): raise RuntimeError('checksum error mismatch')
                else:
                    want=dict(algorithm='SHA-256',file=str(source),bytes=3,sha256=actual,
                              expectedSha256=actual if code==0 else '0'*64,matches=code==0)
                    if value!=want: raise RuntimeError('checksum comparison JSON mismatch')
            count+=1
            return command.stdout
        expected=(actual+'  '+source.name+'\n').encode()
        run(['--checksum',source],text=expected)
        run(['--checksum',source,checksum],text=expected)
        if checksum.read_bytes()!=expected: raise RuntimeError('saved checksum text mismatch')
        run(['--compare-file',source,checksum])
        for data in (actual,actual.upper(),actual+' *input.bin\r\n','SHA256 (입력 Ω.txt) = '+actual,
                     '\ufeff'+actual+'  input.bin\r\n',' \r\n'+actual+'\n\n'):
            checksum.write_text(data,encoding='utf-8')
            run(['--compare-file',source,checksum])
        checksum.write_text('0'*64+'  other.bin\n',encoding='utf-8')
        stdout=run(['--compare-file',source,checksum,report],2)
        if report.read_bytes()!=stdout: raise RuntimeError('saved mismatch JSON differs')
        before=report.read_bytes()
        for data in (b'',b'bad',b'z'*64,(actual+'  a\n'+actual+'  b\n').encode(),b'x'*65537,
                     (actual+'  a\x00').encode(),actual.encode('utf-16')):
            checksum.write_bytes(data)
            run(['--compare-file',source,checksum,report],1,error=13)
            if report.read_bytes()!=before: raise RuntimeError('invalid checksum damaged prior report')
        run(['--compare-file',root/'missing.bin',checksum],1,error=13)
        run(['--compare-file',source,root/'missing.sha256'],1,error=2)
        run(['--checksum',source,source],1,error=7)
        alias=root/'alias.bin'; alias.hardlink_to(source)
        run(['--checksum',source,alias],1,error=7)
        checksum.write_text(actual,encoding='ascii')
        stdout=run(['--compare-file',source,checksum,report])
        if report.read_bytes()!=stdout or source.read_bytes()!=b'abc' or list(root.glob('*.tmp')):
            raise RuntimeError('checksum publication/source protection failed')
    return dict(ok=True,verification='fairy-sha256-checksum-files',cases=count,guiLaunched=False,inputInjected=False)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe',type=Path,default=Path(__file__).parent/'build/win32-hash.exe')
    args=parser.parse_args()
    try: print(json.dumps(verify(args.exe.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
