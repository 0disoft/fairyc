"""Select validation from a complete Git diff, defaulting to the compiler suite."""

import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys


PREVIEW_VERIFIER = "tools/windows-preview/verify-preview2.ps1"


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


def validation_route(event_name, event, directory=Path(".")):
    try:
        if event_name == "push" and not event["ref"].startswith("refs/tags/"):
            base, head = event["before"], event["after"]
            refs = [base, head]
        elif event_name == "pull_request":
            base = event["pull_request"]["base"]["sha"]
            head = event["pull_request"]["head"]["sha"]
            refs = [f"{base}...{head}"]
        else:
            return "full"
        if not all(re.fullmatch(r"[0-9a-f]{40}", sha) and sha != "0" * 40 for sha in (base, head)):
            return "full"
        result = subprocess.run(
            ["git", "diff", "--no-renames", "--name-status", "-z", *refs, "--"],
            cwd=directory, capture_output=True, check=True, timeout=60,
        )
        if not result.stdout.endswith(b"\0"):
            return "full"
        fields = result.stdout[:-1].split(b"\0")
        if len(fields) % 2:
            return "full"
        route = "docs"
        for status, name in zip(fields[::2], fields[1::2]):
            path = os.fsdecode(name)
            if path == PREVIEW_VERIFIER and status in {b"A", b"M"}:
                route = "windows-preview"
            elif not documentation_path(path):
                return "full"
        return route
    except (KeyError, TypeError, ValueError, OSError, subprocess.SubprocessError):
        # Missing history or an unreadable diff must never suppress compiler checks.
        return "full"


def main():
    route = "full"
    try:
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding="utf-8"))
        route = validation_route(os.environ.get("GITHUB_EVENT_NAME", ""), event)
    except (KeyError, OSError, ValueError):
        pass
    print(f"route={route}")
    print(f"Frontend validation route: {route}", file=sys.stderr)


if __name__ == "__main__":
    main()
