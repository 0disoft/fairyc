"""Skip compiler checks only when a complete Git diff contains known documentation paths."""

import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys


def documentation_path(name):
    path = PurePosixPath(name)
    parts = path.parts
    if not parts:
        return False
    if path.suffix == ".md":
        if len(parts) == 1 or parts[0] in {"docs", "diagrams", ".agents"}:
            return True
        if path.name == "README.md" and parts[0] in {"examples", "editors", "spec", "assets"}:
            return True
        if parts[0] in {"examples", "editors"} and {"docs", "guide"}.intersection(parts[1:-1]):
            return True
    if path.suffix.lower() in {".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico"}:
        return parts[0] in {"docs", "diagrams"} or parts[:2] == ("assets", "brand")
    return parts[0] == "diagrams" and path.suffix == ".mmd"


def needs_checks(event_name, event, directory=Path(".")):
    try:
        if event_name == "push" and not event["ref"].startswith("refs/tags/"):
            base, head = event["before"], event["after"]
            refs = [base, head]
        elif event_name == "pull_request":
            base = event["pull_request"]["base"]["sha"]
            head = event["pull_request"]["head"]["sha"]
            refs = [f"{base}...{head}"]
        else:
            return True
        if not all(re.fullmatch(r"[0-9a-f]{40}", sha) and sha != "0" * 40 for sha in (base, head)):
            return True
        result = subprocess.run(
            ["git", "diff", "--no-renames", "--name-only", "-z", *refs, "--"],
            cwd=directory, capture_output=True, check=True, timeout=60,
        )
        paths = [os.fsdecode(name) for name in result.stdout.split(b"\0") if name]
        return not paths or not all(documentation_path(name) for name in paths)
    except (KeyError, TypeError, ValueError, OSError, subprocess.SubprocessError):
        # Missing history or an unreadable diff must never suppress compiler checks.
        return True


def main():
    run = True
    try:
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding="utf-8"))
        run = needs_checks(os.environ.get("GITHUB_EVENT_NAME", ""), event)
    except (KeyError, OSError, ValueError):
        pass
    print(f"run_tests={'true' if run else 'false'}")
    print("Compiler checks required." if run else "Documentation-only change; compiler checks skipped.", file=sys.stderr)


if __name__ == "__main__":
    main()
