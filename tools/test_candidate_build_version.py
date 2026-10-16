"""Narrow regression checks for candidate version generation."""
import json
import subprocess
import tempfile
import unittest
from pathlib import Path

from candidate_build_version import render, source_state


class VersionGenerationTests(unittest.TestCase):
    def test_replace_only_version_and_escape_source_literal(self):
        template = 'module public_cli_data;\npub fn ref str help(){return "help";}\n' + \
                   'pub fn ref str version(){return "old\\n";}\n'
        for compiler, revision, modified in (
                ('v0.0.0-20261016000000-0123456789ab+dirty', '0123456789abcdef', 'true'),
                ('v1.2.3', '0123456789abcdef', 'false'),
                ('development', 'unknown', 'unknown')):
            text = f'Fairy C Stage 0\ncompiler: {compiler}\nsdk: sdk 1\nlanguage: lang/1\nruntime: abi/1\nrevision: {revision}\nmodified: {modified}\n'
            got = render(template, text)
            self.assertEqual(got, template.replace('"old\\n"', json.dumps(text)))
        self.assertIn('\\"', render(template, text.replace('development', 'a"b')))

    def test_reject_missing_duplicate_function_and_invalid_report(self):
        template = 'pub fn ref str version(){return "old";}'
        valid = 'Fairy C Stage 0\ncompiler: development\nsdk: sdk 1\nlanguage: lang/1\nruntime: abi/1\nrevision: unknown\nmodified: unknown\n'
        for source, text in (('', valid), (template * 2, valid),
                             (template, valid.replace('unknown\n', 'maybe\n')),
                             (template, valid.replace('compiler: development', 'compiler: ')),
                             (template, valid + 'extra\n'), (template, valid.rstrip())):
            with self.assertRaises(ValueError):
                render(source, text)

    def test_source_snapshot_detects_untracked_changes_and_tags(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def git(*args):
                subprocess.run(['git', *args], cwd=root, check=True, capture_output=True)
            git('init')
            (root / 'tracked').write_text('source')
            git('add', 'tracked')
            git('-c', 'user.name=Version Test', '-c', 'user.email=version@example.invalid',
                '-c', 'commit.gpgsign=false', 'commit', '-m', 'source')
            clean = source_state(root)
            self.assertEqual(clean[1:], (b'', b''))
            (root / 'untracked').write_text('dirty')
            self.assertNotEqual(source_state(root), clean)
            (root / 'untracked').unlink()
            git('tag', 'v1.2.3')
            self.assertNotEqual(source_state(root), clean)


if __name__ == '__main__':
    unittest.main()
