"""Exercise comparison JSON, exit codes and save guards without opening a GUI."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile


def verify(image):
    count=0
    with tempfile.TemporaryDirectory(prefix='fairy-비교-') as directory:
        root=Path(directory); source=root/'입력 Ω.txt'; source.write_bytes(b'abc')
        actual=hashlib.sha256(b'abc').hexdigest()
        def run(expected,target=None,error=None,path=source):
            nonlocal count
            command=subprocess.run([str(image),'--compare',str(path),expected]+([str(target)] if target else []),capture_output=True,timeout=30)
            if command.stderr: raise RuntimeError('comparison wrote unexpected stderr')
            reply=json.loads(command.stdout)
            if error is not None:
                if command.returncode!=1 or reply!=dict(ok=False,errorCode=error): raise RuntimeError(f'comparison error mismatch: {reply}')
            else:
                normalized=expected.strip().lower(); matches=normalized==actual
                want=dict(algorithm='SHA-256',file=str(path),bytes=3,sha256=actual,expectedSha256=normalized,matches=matches)
                if command.returncode!=(0 if matches else 2) or reply!=want: raise RuntimeError('comparison JSON/exit code mismatch')
                if target and target.read_bytes()!=command.stdout: raise RuntimeError('saved comparison report differs from stdout')
            count+=1
        for expected in (actual,actual.upper(),' \t'+actual+'\r\n','0'*64): run(expected)
        target=root/'결과.json'
        run(actual,target); run('0'*64,target)
        for invalid in ('','f'*63,'f'*65,'z'*64,actual[:32]+' '+actual[32:],'가'*64): run(invalid,error=12)
        before=target.read_bytes(); run('bad',target,error=12)
        if target.read_bytes()!=before: raise RuntimeError('invalid comparison damaged prior report')
        run(actual,path=root/'missing.bin',error=2)
        run('bad',path=root/'missing.bin',error=12)
        run(actual,source,error=7)
        if source.read_bytes()!=b'abc' or list(root.glob('*.tmp')): raise RuntimeError('source changed or temporary report leaked')
    return dict(ok=True,verification='fairy-sha256-comparison',cases=count,guiLaunched=False,inputInjected=False)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe',type=Path,default=Path(__file__).parent/'build/win32-hash.exe')
    args=parser.parse_args()
    try: print(json.dumps(verify(args.exe.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
