from pathlib import Path
import subprocess
import tempfile
import unittest

from ci_changes import PREVIEW_VERIFIER, documentation_path, validation_route


class DocumentationPaths(unittest.TestCase):
    def test_prose_and_brand_artwork(self):
        for path in (
            "README.md", "docs/product/02-spec.md", ".agents/skills/feature/SKILL.md",
            "examples/struct_array_quickstart/README.md", "spec/fixtures/c-abi/README.md",
            "editors/vscode/README.md", "examples/lexer_candidate/guide/07-usage.md",
            "assets/brand/README.md", "assets/brand/logo.png", "docs/images/example.svg",
            "diagrams/frontend.mmd",
        ):
            with self.subTest(path=path):
                self.assertTrue(documentation_path(path))

    def test_executable_inputs_and_unknown_paths_require_checks(self):
        for path in (
            "internal/compiler/check.go", "internal/testdata/README.md", "internal/testdata/input.png",
            "examples/demo/src/main.fy", "spec/grammar/stage0.ebnf", "spec/fixtures/semantic/manifest-v0.json",
            "docs/ops/evidence/result.json", "editors/vscode/package.json", "editors/zed/Cargo.lock",
            "go.mod", "go.sum", ".github/workflows/frontend.yml", ".github/scripts/ci_changes.py",
            "new-component/input.md", "assets/runtime/icon.png",
        ):
            with self.subTest(path=path):
                self.assertFalse(documentation_path(path))


class GitDiffRouting(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.git("init", "-q")
        self.git("config", "user.name", "CI routing tests")
        self.git("config", "user.email", "ci-routing-tests@localhost")
        self.git("config", "commit.gpgsign", "false")
        self.write("README.md", "Initial documentation\n")
        self.base = self.commit()

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.root, stderr=subprocess.PIPE).decode().strip()

    def write(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def commit(self):
        self.git("add", "--all")
        self.git("commit", "-qm", "Routing fixture")
        return self.git("rev-parse", "HEAD")

    def push(self, base, head):
        return validation_route("push", {"ref": "refs/heads/main", "before": base, "after": head}, self.root)

    def pull_request(self, base, head):
        return validation_route("pull_request", {"pull_request": {
            "base": {"sha": base}, "head": {"sha": head},
        }}, self.root)

    def test_docs_only_push_and_pull_request(self):
        self.write("docs/with spaces.md", "Documentation\n")
        head = self.commit()
        self.assertEqual(self.push(self.base, head), "docs")
        self.assertEqual(self.pull_request(self.base, head), "docs")

    def test_preview_verifier_with_docs_routes_push_and_pull_request_to_preview(self):
        self.write(PREVIEW_VERIFIER, "Initial verifier\n")
        base = self.commit()
        self.write(PREVIEW_VERIFIER, "Changed verifier\n")
        self.write("docs/ops/preview.md", "Preview guide\n")
        head = self.commit()
        self.assertEqual(self.push(base, head), "windows-preview")
        self.assertEqual(self.pull_request(base, head), "windows-preview")

    def test_preview_with_source_or_workflow_changes_requires_full_checks(self):
        self.write(PREVIEW_VERIFIER, "Preview verifier\n")
        preview = self.commit()
        self.assertEqual(self.push(self.base, preview), "windows-preview")
        for name in ("internal/compiler/change.go", ".github/workflows/frontend.yml"):
            with self.subTest(name=name):
                self.git("checkout", "--detach", preview)
                self.write(name, "Test input\n")
                self.assertEqual(self.push(self.base, self.commit()), "full")

    def test_removing_preview_verifier_requires_full_checks(self):
        self.write(PREVIEW_VERIFIER, "Preview verifier\n")
        base = self.commit()
        self.git("rm", PREVIEW_VERIFIER)
        self.assertEqual(self.push(base, self.commit()), "full")

    def test_complete_push_not_just_latest_commit_or_first_300_files(self):
        self.write("internal/compiler/change.go", "package compiler\n")
        self.commit()
        for number in range(301):
            self.write(f"docs/page-{number:03}.md", "Documentation\n")
        head = self.commit()
        self.assertEqual(self.push(self.base, head), "full")

    def test_rename_out_of_code_still_requires_checks(self):
        self.write("internal/input.go", "package input\n")
        base = self.commit()
        self.git("mv", "internal/input.go", "README.md.old")
        self.git("mv", "README.md.old", "docs-moved.md")
        self.assertEqual(self.push(base, self.commit()), "full")

    def test_pr_uses_merge_base_not_unrelated_base_branch_changes(self):
        self.write("docs/topic.md", "Topic prose\n")
        head = self.commit()
        self.git("checkout", "--detach", self.base)
        self.write("internal/base-only.go", "package base\n")
        base = self.commit()
        self.assertEqual(self.pull_request(base, head), "docs")

    def test_unknown_history_initial_push_manual_and_tags_run_checks(self):
        self.assertEqual(self.push("f" * 40, self.base), "full")
        self.assertEqual(self.push("0" * 40, self.base), "full")
        self.assertEqual(self.push(self.base, self.base), "full")
        self.assertEqual(validation_route("workflow_dispatch", {}, self.root), "full")
        self.assertEqual(validation_route("push", {"ref": "refs/tags/v1"}, self.root), "full")
        self.assertEqual(validation_route("pull_request", {}, self.root), "full")


if __name__ == "__main__":
    unittest.main()
