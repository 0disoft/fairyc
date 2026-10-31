import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from windows_mixed_stack_evidence import CASES, HASHES, collect


def fixture():
    rows = []
    for name in sorted(CASES):
        for profile in ("low", "optimized"):
            success = name == "walk-and-c-boundary-catch"
            early = name == "callback-trap-no-unwind"
            def event(kind, *values):
                return {"kind": kind, "values": list(values)}
            trace = [event("callback", 1000, 13 if early else 40, 1)]
            if not early:
                trace += [event("frame", stage, 2000 + stage, 4000 + stage * 32) for stage in range(3)]
                trace += [event("handler", 5000, 5001 if name == "reject-handler-address" else 5000,
                                6000, 6004 if name == "reject-handler-data" else 6000,
                                7000, 7000, 8000, 8008 if name == "reject-caller-stack" else 8000, 9000)]
                if success:
                    trace.append(event("frame", 3, 7000, 8000))
                trace.append(event("result", 41 if success else -99, 1, 1, 1))
            row = {
                "case": name, "profile": profile, "toolchain": "fixture-msvc",
                "runtimeFunctions": 434, "exit": 42 if success else 134,
                "trapCode": 0 if success else 6, "trapCategory": "" if success else "assertion-failed",
                "trapControlHex": "" if success else "06", "trace": trace,
                **{key: "a" * 64 for key in HASHES},
            }
            rows.append(row)
    return {"schema": "fairyc.windows-mixed-stack/v1", "sourceRevision": "a" * 40, "passed": True, "rows": rows}


class MixedStackReceipt(unittest.TestCase):
    def setUp(self):
        self.env = {
            "SOURCE_SHA": "a" * 40, "WORKFLOW_SHA": "b" * 40,
            "CANDIDATE_SHA": "a" * 40, "SMOKE_OUTCOME": "success",
            "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "2",
        }

    def collect(self, payload, env=None, revision=None):
        return collect(json.dumps(payload).encode(), env or self.env, revision or "a" * 40)

    def test_full_matrix_binds_all_hashes_and_native_rows(self):
        report = self.collect(fixture())
        self.assertTrue(report["passed"], report)
        self.assertEqual(report["rows"], fixture()["rows"])
        self.assertEqual(len(report["evidenceSHA256"]), 64)

    def test_cli_checks_real_head_and_writes_log_and_compact_summary(self):
        root = Path(__file__).resolve().parents[2]
        actual = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        payload = fixture()
        payload["sourceRevision"] = actual
        with tempfile.TemporaryDirectory(prefix="fairyc-mixed-ci-test-") as directory:
            receipt, summary = Path(directory) / "receipt.json", Path(directory) / "summary.md"
            receipt.write_text(json.dumps(payload), encoding="utf-8")
            env = {**os.environ, **self.env, "SOURCE_SHA": actual, "CANDIDATE_SHA": actual,
                   "GITHUB_STEP_SUMMARY": str(summary)}
            command = [sys.executable, "-B", str(root / ".github/scripts/windows_mixed_stack_evidence.py"), str(receipt)]
            result = subprocess.run(command, cwd=root, env=env, capture_output=True, text=True, timeout=15)
            self.assertEqual(result.returncode, 0, result.stderr)
            report = json.loads(result.stdout)
            self.assertTrue(report["passed"])
            self.assertEqual(report["sourceRevision"], actual)
            self.assertEqual(len(report["rows"]), 10)
            text = summary.read_text(encoding="utf-8")
            self.assertIn(report["evidenceSHA256"], text)
            self.assertNotIn('"trace"', text)
            receipt.unlink()
            result = subprocess.run(command, cwd=root, env=env, capture_output=True, text=True, timeout=15)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(json.loads(result.stdout)["passed"])

    def test_revision_and_run_identity_fail_closed(self):
        for field, value in (
            ("SOURCE_SHA", "c" * 40), ("WORKFLOW_SHA", ""), ("CANDIDATE_SHA", "d" * 40),
            ("SMOKE_OUTCOME", "failure"), ("SMOKE_OUTCOME", "skipped"),
            ("GITHUB_RUN_ID", ""), ("GITHUB_RUN_ATTEMPT", "0"),
        ):
            with self.subTest(field=field, value=value):
                env = {**self.env, field: value}
                self.assertFalse(self.collect(fixture(), env)["passed"])
        self.assertFalse(self.collect(fixture(), revision="e" * 40)["passed"])

    def test_missing_stale_or_failed_receipt(self):
        for field, value in (("sourceRevision", "f" * 40), ("schema", "old"), ("passed", False)):
            payload = fixture()
            payload[field] = value
            self.assertFalse(self.collect(payload)["passed"])
        for raw in (b"", b"{", b"null", b"[]"):
            self.assertFalse(collect(raw, self.env, "a" * 40)["passed"])

    def test_native_evidence_mutations_fail_closed(self):
        payload = fixture()
        success = next(row for row in payload["rows"] if row["case"] == "walk-and-c-boundary-catch")
        trap = next(row for row in payload["rows"] if row["case"] == "callback-trap-no-unwind")
        mutations = [
            lambda p: p["rows"].pop(),
            lambda p: p["rows"].__setitem__(0, copy.deepcopy(p["rows"][1])),
        ]
        for row, field, value in (
            (success, "imageSHA256", "bad"), (success, "exit", 0),
            (success, "profile", "other"), (success, "runtimeFunctions", 0),
            (trap, "trapCode", 5), (trap, "trapControlHex", ""),
            (trap, "trapCategory", "bounds"),
            (success, "trace", success["trace"][:-1]),
        ):
            index = payload["rows"].index(row)
            mutations.append(lambda p, i=index, f=field, v=value: p["rows"][i].__setitem__(f, v))
        index = payload["rows"].index(success)
        for event_index, value_index in ((0, 1), (1, 0), (2, 2), (4, 0), (4, 2), (4, 4), (4, 6), (6, 0), (6, 1)):
            def mutate(p, e=event_index, v=value_index):
                p["rows"][index]["trace"][e]["values"][v] = -1
            mutations.append(mutate)
        for number, mutation in enumerate(mutations):
            with self.subTest(number=number):
                changed = copy.deepcopy(payload)
                mutation(changed)
                self.assertFalse(self.collect(changed)["passed"])


if __name__ == "__main__":
    unittest.main()
