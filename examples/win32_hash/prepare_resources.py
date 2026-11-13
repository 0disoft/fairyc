"""Prepare configured icon/version resources without bumping the version."""
from pathlib import Path
import sys
ROOT = Path(__file__).resolve().parent
if not (ROOT/"win32_app").is_dir(): sys.path.insert(0,str(ROOT.parents[1]/"tools"))
from win32_app import resources
ICON_ID = resources.ICON_ID
def metadata(root=ROOT): return resources.metadata(root)
def resource_text(root=ROOT): return resources.resource_text(root)
def prepare(root=ROOT): return resources.prepare(root)
if __name__ == "__main__":
    prepare(); print("Windows icon/version resources prepared; package version unchanged")
