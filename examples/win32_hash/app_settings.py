"""Fairy Hash defaults for shared Win32 app identity tooling."""
from pathlib import Path
import sys
ROOT = Path(__file__).resolve().parent
if not (ROOT/"win32_app").is_dir(): sys.path.insert(0,str(ROOT.parents[1]/"tools"))
from win32_app.settings import App,resource_string
from win32_app.settings import load as shared_load
def load(root=ROOT): return shared_load(root)
