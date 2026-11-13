"""Prepare configured resources; never increment the package version."""
from pathlib import Path
from win32_app.resources import prepare
ROOT=Path(__file__).resolve().parent
if __name__=='__main__': prepare(ROOT)
