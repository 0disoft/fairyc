import io
import json
import unittest

from linux_phase2_evidence import CASES, ROOTS, collect


def events(action="pass"):
    return [{"Package": p, "Test": t, "Action": action} for p, t in sorted(ROOTS | CASES)]


class LinuxPhase2Evidence(unittest.TestCase):
    def run_events(self, items):
        output = io.StringIO()
        report = collect((json.dumps(item) for item in items), output)
        return report, output.getvalue()

    def test_requires_all_native_cases_and_parent_results(self):
        report, _ = self.run_events(events())
        self.assertTrue(report["complete"])
        self.assertEqual(len(report["passedCases"]), 12)
        for missing in (next(iter(ROOTS)), next(iter(CASES))):
            with self.subTest(missing=missing):
                report, _ = self.run_events([
                    e for e in events() if (e["Package"], e["Test"]) != missing
                ])
                self.assertFalse(report["complete"])
                self.assertEqual(len(report["missing"]), 1)

    def test_skip_failure_duplicate_and_unrelated_failure_reject_receipt(self):
        package, test = sorted(CASES)[0]
        for action in ("skip", "fail", "pass"):
            with self.subTest(action=action):
                report, _ = self.run_events(events() + [
                    {"Package": package, "Test": test, "Action": action}
                ])
                self.assertFalse(report["complete"])
        for action in ("fail", "build-fail"):
            report, _ = self.run_events(events() + [{"Package": "other", "Action": action}])
            self.assertFalse(report["complete"])

    def test_keeps_selected_output_and_failure_tail_without_success_noise(self):
        package, test = sorted(CASES)[0]
        report, output = self.run_events([
            {"Package": package, "Test": test, "Action": "output", "Output": "native proof\n"},
            {"Package": "other", "Test": "TestOK", "Action": "output", "Output": "noise\n"},
            {"Package": "other", "Test": "TestOK", "Action": "pass"},
            {"Package": "other", "Test": "TestBad", "Action": "output", "Output": "bad assertion\n"},
            {"Package": "other", "Test": "TestBad", "Action": "fail"},
        ] + events())
        self.assertFalse(report["complete"])
        self.assertIn("native proof", output)
        self.assertIn("bad assertion", output)
        self.assertNotIn("noise", output)

    def test_empty_malformed_and_wrong_identity_cannot_pass(self):
        for lines in ([], ["not-json"], ["[]"], [json.dumps({"Action": "pass", "Package": "wrong"})]):
            with self.subTest(lines=lines):
                self.assertFalse(collect(lines, io.StringIO())["complete"])


if __name__ == "__main__":
    unittest.main()
