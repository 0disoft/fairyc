"""Validate portable resources and the Hash-specific CLI operations."""
from prepare_resources import ROOT
from win32_app.portable import verify as shared_verify
from verify_cli import verify as cli
from verify_compare import verify as compare
from verify_checksum import verify as checksum
import argparse,json,subprocess,zipfile
from pathlib import Path
def checks(image):
    return dict(cliCases=cli(image)["cases"],comparisonCases=compare(image)["cases"],checksumCases=checksum(image)["cases"])
def verify(archive,root=ROOT):
    result=shared_verify(archive,root,checks); result["verification"]="fairy-hash-portable-package"; return result
if __name__ == "__main__":
    parser=argparse.ArgumentParser(description=__doc__); parser.add_argument("archive",type=Path)
    try: print(json.dumps(verify(parser.parse_args().archive.resolve())))
    except (OSError,ValueError,RuntimeError,zipfile.BadZipFile,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
