"""Keep the Hash reference's vendored worker sources aligned with the source bundle."""
import argparse
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/'modules/win32_jobs'

def files():
    return sorted(p for directory in ('src','native') for p in (SOURCE/directory).rglob('*') if p.is_file())

def install(target):
    for source in files():
        output=target/source.relative_to(SOURCE);output.parent.mkdir(parents=True,exist_ok=True)
        output.write_bytes(source.read_bytes().replace(b'\r\n',b'\n'))

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--check',action='store_true');args=parser.parse_args()
    target=ROOT/'examples/win32_hash'
    if args.check:
        changed=[str(p.relative_to(SOURCE)) for p in files() if not (target/p.relative_to(SOURCE)).is_file() or
                 (target/p.relative_to(SOURCE)).read_bytes().replace(b'\r\n',b'\n')!=p.read_bytes().replace(b'\r\n',b'\n')]
        if changed: raise SystemExit('worker snapshots are stale: '+', '.join(changed))
        print('Win32 worker snapshots match canonical sources')
    else: install(target);print('Win32 worker snapshots refreshed')
