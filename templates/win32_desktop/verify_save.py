"""Exercise the actual shared publication controller without a window."""
import json
from pathlib import Path
import os
import subprocess
import tempfile

def verify(image):
    count=0
    with tempfile.TemporaryDirectory(prefix='save-한글-') as directory:
        root=Path(directory);target=root/'결과 Ω.txt'
        def save(path,text,options=(),error=0):
            nonlocal count
            old_temps={p:p.read_bytes() for p in root.glob('*.fairy-save-*.tmp')}
            result=subprocess.run([str(image),'--save',str(path),text,*map(str,options)],capture_output=True,timeout=20)
            expected=0 if not error else 2 if error==8 else 1
            if result.returncode!=expected or result.stderr or json.loads(result.stdout)!={'ok':not bool(error),'errorCode':error}:
                raise RuntimeError(f'save failed: {result.returncode} {result.stdout!r} {result.stderr!r}')
            if {p:p.read_bytes() for p in root.glob('*.fairy-save-*.tmp')}!=old_temps:
                raise RuntimeError('publication leaked or modified an unowned temporary file')
            count+=1
        save(target,'한글 Ω\r\nresult');assert target.read_bytes()=='한글 Ω\r\nresult'.encode()
        save(target,'replacement');assert target.read_bytes()==b'replacement'
        save(target,'');assert target.read_bytes()==b''
        target.write_bytes(b'old')
        save(target,'cancelled',['--cancel-before-commit'],8);assert target.read_bytes()==b'old'
        absent=root/'absent.txt';save(absent,'cancelled',['--cancel-before-commit'],8);assert not absent.exists()
        save(root/'missing'/'result.txt','failed',error=6);assert not (root/'missing').exists()
        folder=root/'folder';folder.mkdir();save(folder,'failed',error=6);assert folder.is_dir()
        collision=Path(str(target)+'.fairy-save-0.tmp');collision.write_bytes(b'unowned')
        save(target,'after collision');assert target.read_bytes()==b'after collision' and collision.read_bytes()==b'unowned'
        for i in range(1,32): Path(str(target)+f'.fairy-save-{i}.tmp').write_bytes(b'unowned')
        save(target,'exhausted',error=6);assert target.read_bytes()==b'after collision'
        for i in range(32): Path(str(target)+f'.fairy-save-{i}.tmp').unlink()
        source=root/'source.bin';source.write_bytes(b'original')
        save(source,'overwrite',['--protect-source',source],7);assert source.read_bytes()==b'original'
        alias=root/'alias.bin';os.link(source,alias)
        save(alias,'overwrite',['--protect-source',source],7);assert source.read_bytes()==alias.read_bytes()==b'original'
        save(target,'protected',['--protect-source',source]);assert source.read_bytes()==b'original' and target.read_bytes()==b'protected'
    return count
