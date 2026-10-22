"""Stream Go test results and require the Phase 2 Linux native cases, without reruns."""

import json
import sys
from collections import defaultdict, deque


ROOTS = {
    ("github.com/0disoft/fairyc/internal/cli", "TestLinuxFileProgressNative"),
    ("github.com/0disoft/fairyc/internal/cstdlib", "TestSystemHeapOOMNative"),
    ("github.com/0disoft/fairyc/internal/cstdlib", "TestLinuxPreparedPublication"),
    ("github.com/0disoft/fairyc/internal/cli", "TestBytesEqualExecution"),
    ("github.com/0disoft/fairyc/internal/cli", "TestBytesCompareExecution"),
    ("github.com/0disoft/fairyc/internal/cli", "TestCheckedI64DecimalExecution"),
    ("github.com/0disoft/fairyc/internal/cruntime", "TestLinuxTrapReceiptProgressNative"),
    ("github.com/0disoft/fairyc/internal/compiler", "TestPrivateVectorNestedResourceDrops"),
}
CASES = {
    (package, f"{test}/{compiler}/{profile}")
    for package, test in ROOTS
    for compiler in ("gcc", "clang")
    for profile in ("low", "optimized")
}


def collect(lines, output=sys.stdout):
    passed, skipped, errors = set(), set(), []
    tails = defaultdict(lambda: deque(maxlen=80))
    for line in lines:
        try:
            event = json.loads(line)
            if not isinstance(event, dict):
                raise ValueError("expected a Go test event object")
        except ValueError as error:
            errors.append(f"invalid test event: {error}")
            continue
        package, test = event.get("Package", ""), event.get("Test", "")
        key = package, test
        selected = (package, test.split("/")[0]) in ROOTS
        action = event.get("Action")
        if "Output" in event:
            if selected or not test:
                print(event["Output"], end="", file=output, flush=True)
            else:
                tails[key].append(event["Output"])
        if action in ("fail", "build-fail"):
            errors.append(f"failed: {package}::{test}")
            if not selected:
                print("".join(tails[key]), end="", file=output, flush=True)
        if selected and action == "skip":
            skipped.add(key)
        if selected and action == "pass":
            if key in passed:
                errors.append(f"duplicate pass: {package}::{test}")
            passed.add(key)
        if action in ("pass", "skip", "fail"):
            tails.pop(key, None)
    missing = (ROOTS | CASES) - passed
    report = {
        "schema": "fairyc.phase2-linux-evidence/v4",
        "complete": not (errors or skipped or missing),
        "passedCases": sorted(f"{p}::{t}" for p, t in passed & CASES),
        "missing": sorted(f"{p}::{t}" for p, t in missing),
        "skipped": sorted(f"{p}::{t}" for p, t in skipped),
        "errors": errors,
    }
    print(json.dumps(report, sort_keys=True), file=output, flush=True)
    return report


if __name__ == "__main__":
    sys.exit(0 if collect(sys.stdin)["complete"] else 1)
