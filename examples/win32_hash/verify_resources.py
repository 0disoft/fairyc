"""Inspect configured PE resources without running the application."""
from pathlib import Path
import sys
ROOT = Path(__file__).resolve().parent
if not (ROOT/"win32_app").is_dir(): sys.path.insert(0,str(ROOT.parents[1]/"tools"))
import argparse,json,struct
from win32_app.pe import verify as shared_verify
def verify(image,root=ROOT):
    result=shared_verify(image,root); result["verification"]="fairy-hash-pe-resources"; return result
if __name__ == "__main__":
    parser=argparse.ArgumentParser(description=__doc__); parser.add_argument("--exe",type=Path,default=ROOT/"build/win32-hash.exe")
    try: print(json.dumps(verify(parser.parse_args().exe.resolve())))
    except (OSError,ValueError,RuntimeError,struct.error) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
