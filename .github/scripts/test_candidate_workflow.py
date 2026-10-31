"""Exercise the workflow's actual guards and PowerShell blocks without a runner."""

import ast
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import textwrap
import unittest


ROOT = Path(__file__).resolve().parents[2]
WORKFLOW = (ROOT / ".github/workflows/frontend.yml").read_text(encoding="utf-8")
PWSH = shutil.which("pwsh")


def diagnostic_text(output):
    # PowerShell's error view can color and wrap one message across lines.
    # Keep the words intact while ignoring terminal presentation in assertions.
    return " ".join(re.sub(r"\x1b\[[0-?]*[ -/]*[@-~]", "", output).split())


class PowerShellDiagnostics(unittest.TestCase):
    def test_colored_wrapped_revision_error_keeps_the_expected_message(self):
        # The Linux runner wrapped between "differs" and "from" in this error.
        output = ("\x1b[31;1mException: \x1b[0m/tmp/revision.ps1:6\n"
                  "\x1b[31;1mChecked-out revision " + "4" * 40 + " differs\x1b[0m\n"
                  "\x1b[31;1mfrom requested candidate " + "a" * 40 + "\x1b[0m\n")
        expected = "differs from requested candidate"
        self.assertNotIn(expected, output)
        self.assertIn(expected, diagnostic_text(output))
        self.assertEqual(diagnostic_text("candidate_sha must be\r\n  a full SHA"),
                         "candidate_sha must be a full SHA")
        self.assertNotIn(expected, diagnostic_text("Cannot resolve checked-out source revision"))


def job_block(name):
    return re.search(rf"^  {re.escape(name)}:\n(.*?)(?=^  [\w-]+:|\Z)",
                     WORKFLOW, re.M | re.S).group(1)


def step_script(step):
    # Extract only literal run blocks at the known step indentation, not general YAML.
    block = re.search(rf"^        id: {re.escape(step)}\n(.*?)(?=^      - |\Z)",
                      job_block("windows-native"), re.M | re.S).group(1)
    return textwrap.dedent(re.search(r"^        run: \|\n((?:          .*\n)+)", block, re.M).group(1))


def permits(job, event, candidate="", changes="success", run_tests="true", cancelled=False):
    expression = re.search(r"^    if: \$\{\{ (.+) \}\}$", job_block(job), re.M).group(1)
    values = {"github.event_name": event, "inputs.candidate_sha": candidate,
              "needs.changes.result": changes, "needs.changes.outputs.run_tests": run_tests,
              "cancelled()": cancelled}
    for name, value in values.items():
        expression = expression.replace(name, repr(value))
    expression = re.sub(r"!(?!=)", " not ", expression).replace("&&", " and ").replace("||", " or ").strip()
    tree = ast.parse(expression, mode="eval")
    allowed = (ast.Expression, ast.BoolOp, ast.UnaryOp, ast.Compare, ast.Constant,
               ast.And, ast.Or, ast.Not, ast.Eq, ast.NotEq)
    if any(not isinstance(node, allowed) for node in ast.walk(tree)):
        raise ValueError("Unsupported workflow predicate in local test")
    return eval(compile(tree, "<workflow predicate>", "eval"), {"__builtins__": {}})


class CandidateRouting(unittest.TestCase):
    def test_candidate_only_enables_windows_native_even_when_changes_is_skipped(self):
        for candidate in ("1" * 40, "invalid-input"):
            enabled = {job for job in ("changes", "validate", "windows-native", "windows-parser", "native-c")
                       if permits(job, "workflow_dispatch", candidate, changes="skipped", run_tests="")}
            self.assertEqual(enabled, {"windows-native"})

    def test_regular_events_and_empty_manual_input_preserve_full_guards(self):
        for event in ("push", "pull_request", "workflow_dispatch"):
            for job in ("changes", "validate", "windows-native", "windows-parser", "native-c"):
                self.assertTrue(permits(job, event))
        for job in ("validate", "windows-native", "windows-parser"):
            self.assertFalse(permits(job, "push", run_tests="false"))
            self.assertTrue(permits(job, "push", changes="failure", run_tests=""))
            self.assertFalse(permits(job, "workflow_dispatch", "1" * 40, cancelled=True))


@unittest.skipUnless(PWSH, "PowerShell 7 is required to execute Windows workflow blocks locally")
class CandidateIdentity(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="fairyc-ci-candidate-")
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)

    def execute(self, step, **environment):
        script = self.directory / f"{step}.ps1"
        script.write_text("$ErrorActionPreference = 'Stop'\n" + step_script(step), encoding="utf-8")
        env = {**os.environ, "CANDIDATE_SHA": "", "GITHUB_OUTPUT": str(self.directory / "output"),
               "GITHUB_STEP_SUMMARY": str(self.directory / "summary"), **environment}
        return subprocess.run([PWSH, "-NoProfile", "-NonInteractive", "-File", str(script)],
                              cwd=ROOT, env=env, capture_output=True, encoding="utf-8", timeout=15)

    def test_sha_validation_rejects_refs_short_uppercase_zero_and_shell_text(self):
        self.assertEqual(self.execute("candidate-input", CANDIDATE_SHA="a" * 40).returncode, 0)
        for invalid in ("main", "refs/tags/v1", "a" * 7, "A" * 40, "0" * 40,
                        "a" * 40 + "\n", "$(throw 'input was executed')"):
            with self.subTest(candidate=invalid):
                result = self.execute("candidate-input", CANDIDATE_SHA=invalid)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("candidate_sha must be", diagnostic_text(result.stderr))

    def test_actual_checkout_must_match_and_is_written_to_output(self):
        actual = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
        result = self.execute("revision", CANDIDATE_SHA=actual)
        self.assertEqual(result.returncode, 0, result.stderr)
        output = self.directory / "output"
        self.assertEqual(output.read_text(encoding="utf-8").strip(), f"source_sha={actual}")
        output.unlink()
        mismatch = ("a" if actual[0] != "a" else "b") + actual[1:]
        result = self.execute("revision", CANDIDATE_SHA=mismatch)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("differs from requested candidate", diagnostic_text(result.stderr))
        self.assertFalse(output.exists())

    def test_receipt_distinguishes_source_workflow_and_failed_or_skipped_smoke(self):
        for outcome in ("success", "failure", "skipped", "cancelled"):
            with self.subTest(outcome=outcome):
                result = self.execute("receipt", CANDIDATE_SHA="1" * 40, SOURCE_SHA="1" * 40,
                                      WORKFLOW_SHA="2" * 40, SMOKE_OUTCOME=outcome,
                                      GITHUB_RUN_ID="123", GITHUB_RUN_ATTEMPT="2")
                self.assertEqual(result.returncode, 0, result.stderr)
                receipt = json.loads(result.stdout)
                self.assertEqual(receipt["sourceRevision"], "1" * 40)
                self.assertEqual(receipt["workflowRevision"], "2" * 40)
                self.assertEqual(receipt["requestedRevision"], receipt["sourceRevision"])
                self.assertEqual(receipt["runAttempt"], "2")
                self.assertEqual(receipt["smokeOutcome"], outcome)
                self.assertEqual(receipt["passed"], outcome == "success")
                self.assertIn(json.dumps(receipt["sourceRevision"]),
                              (self.directory / "summary").read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
