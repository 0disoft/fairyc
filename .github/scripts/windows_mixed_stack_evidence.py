"""Bind the existing native smoke's mixed-stack receipt to its checked-out commit."""

import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys


CASES = {
    "walk-and-c-boundary-catch", "callback-trap-no-unwind",
    "reject-handler-address", "reject-handler-data", "reject-caller-stack",
}
HASHES = (
    "fairySourceSHA256", "generatedCSHA256", "companionSourceSHA256",
    "imageSHA256", "callerSHA256", "calleeSHA256",
)


def validate_rows(rows):
    if not isinstance(rows, list) or len(rows) != 10:
        raise ValueError("exactly ten mixed-stack executions required")
    seen = set()
    for row in rows:
        key = row["case"], row["profile"]
        if key[0] not in CASES or key[1] not in {"low", "optimized"} or key in seen:
            raise ValueError("unknown or duplicate mixed-stack case")
        seen.add(key)
        if not row["toolchain"] or type(row["runtimeFunctions"]) is not int or row["runtimeFunctions"] <= 0:
            raise ValueError("missing native image/toolchain identity")
        if any(not re.fullmatch(r"[0-9a-f]{64}", row[name]) for name in HASHES):
            raise ValueError("invalid source/object/image digest")
        success = key[0] == "walk-and-c-boundary-catch"
        early = key[0] == "callback-trap-no-unwind"
        expected = (42, 0, "", "") if success else (134, 6, "assertion-failed", None)
        actual = row["exit"], row["trapCode"], row["trapCategory"], row["trapControlHex"]
        if actual[:3] != expected[:3] or (success and actual[3] != ""):
            raise ValueError("wrong exit/trap mapping")
        if not success:
            control = bytes.fromhex(actual[3])
            if not control or control[0] != 6:
                raise ValueError("missing private assertion trap channel")
        kinds = ["callback", "frame", "frame", "frame", "handler", "result"]
        if success:
            kinds.insert(5, "frame")
        if early:
            kinds = ["callback"]
        trace = row["trace"]
        if [event["kind"] for event in trace] != kinds:
            raise ValueError("missing or reordered native trace")
        stage, previous_sp = 0, 0
        for event in trace:
            values, kind = event["values"], event["kind"]
            if not isinstance(values, list) or any(type(v) is not int for v in values):
                raise ValueError("noninteger trace value")
            if kind == "callback":
                if len(values) != 3 or values[0] <= 0 or values[1:] != [13 if early else 40, 1]:
                    raise ValueError("incorrect synchronous callback input")
            elif kind == "frame":
                if len(values) != 3 or values[0] != stage or values[1] <= 0 or values[2] <= previous_sp:
                    raise ValueError("incorrect mixed frame order or stack")
                stage, previous_sp = stage + 1, values[2]
            elif kind == "handler":
                if len(values) != 9 or min(values) <= 0:
                    raise ValueError("missing handler/data/caller addresses")
                deltas = values[1] - values[0], values[3] - values[2], values[7] - values[6]
                want = {
                    "walk-and-c-boundary-catch": (0, 0, 0),
                    "reject-handler-address": (1, 0, 0),
                    "reject-handler-data": (0, 4, 0),
                    "reject-caller-stack": (0, 0, 8),
                }[key[0]]
                if deltas != want or values[4] != values[5]:
                    raise ValueError("wrong handler/data/caller expectations")
            elif values != [41 if success else -99, 1, 1, 1]:
                raise ValueError("callback return or C-contained catch/finally mismatch")
    return rows


def collect(raw, environment, actual_revision):
    report = {
        "schema": "fairyc.windows-mixed-stack-ci/v1", "passed": False,
        "sourceRevision": environment.get("SOURCE_SHA", ""),
        "requestedRevision": environment.get("CANDIDATE_SHA", ""),
        "workflowRevision": environment.get("WORKFLOW_SHA", ""),
        "runId": environment.get("GITHUB_RUN_ID", ""),
        "runAttempt": environment.get("GITHUB_RUN_ATTEMPT", ""),
        "smokeOutcome": environment.get("SMOKE_OUTCOME", ""),
        "evidenceSHA256": hashlib.sha256(raw).hexdigest(),
        "errors": [],
    }
    try:
        source = report["sourceRevision"]
        if not re.fullmatch(r"[0-9a-f]{40}", source) or source == "0" * 40 or actual_revision != source:
            raise ValueError("checked-out source revision mismatch")
        if report["requestedRevision"] and report["requestedRevision"] != source:
            raise ValueError("requested candidate revision mismatch")
        if not re.fullmatch(r"[0-9a-f]{40}", report["workflowRevision"]) or report["workflowRevision"] == "0" * 40:
            raise ValueError("missing workflow revision")
        if any(not re.fullmatch(r"[1-9][0-9]*", report[key]) for key in ("runId", "runAttempt")):
            raise ValueError("missing hosted run identity")
        if report["smokeOutcome"] != "success":
            raise ValueError("native smoke did not succeed")
        payload = json.loads(raw)
        if payload["schema"] != "fairyc.windows-mixed-stack/v1" or payload["passed"] is not True:
            raise ValueError("invalid native receipt schema/status")
        if payload["sourceRevision"] != source:
            raise ValueError("native receipt source revision mismatch")
        report["rows"] = validate_rows(payload["rows"])
        report["passed"] = True
    except (ValueError, KeyError, TypeError, AttributeError) as error:
        report["errors"].append(str(error))
    return report


def main():
    path = Path(sys.argv[1])
    raw = path.read_bytes() if path.is_file() else b""
    revision = subprocess.check_output(["git", "rev-parse", "--verify", "HEAD"], text=True).strip()
    report = collect(raw, os.environ, revision)
    # Full native addresses, stack rows and image/object hashes remain in the job log.
    print(json.dumps(report, sort_keys=True), flush=True)
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        brief = {key: value for key, value in report.items() if key != "rows"}
        brief["executions"] = len(report.get("rows", []))
        with open(summary, "a", encoding="utf-8") as stream:
            stream.write("\n### Same-commit Fairy/C mixed stack\n\n```json\n")
            stream.write(json.dumps(brief, indent=2) + "\n```\n")
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
