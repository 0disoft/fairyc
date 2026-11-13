"""Install/check the shared UI source snapshot without using the desktop."""
import argparse
from sync_win32_jobs import ROOT,files,install as install_bundle
SOURCE=ROOT/'modules/win32_ui'
def install(target): install_bundle(target,SOURCE)
if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--check',action='store_true');args=parser.parse_args()
    target=ROOT/'examples/win32_hash'
    if args.check:
        changed=[str(p.relative_to(SOURCE)) for p in files(SOURCE) if not (target/p.relative_to(SOURCE)).is_file() or
                 (target/p.relative_to(SOURCE)).read_bytes().replace(b'\r\n',b'\n')!=p.read_bytes().replace(b'\r\n',b'\n')]
        if changed: raise SystemExit('UI snapshots are stale: '+', '.join(changed))
        print('Win32 UI snapshots match canonical sources')
    else: install(target);print('Win32 UI snapshots refreshed')
