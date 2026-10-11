"""Regression tests for GitHub repository URL canonicalization."""

import unittest

from collect_open_source_licenses import normalize_git_url


class GitURLNormalizationTests(unittest.TestCase):
    def test_real_github_remotes(self):
        cases = {
            "git://github.com/org/repo.git": "https://github.com/org/repo.git",
            "http://github.com/org/repo.git": "https://github.com/org/repo.git",
            "git+git://github.com/org/repo.git": "https://github.com/org/repo.git",
            "github:org/repo": "https://github.com/org/repo",
            "git://github.com/org/repo?x=1#readme": "https://github.com/org/repo?x=1#readme",
        }
        for source, expected in cases.items():
            with self.subTest(source=source):
                self.assertEqual(normalize_git_url(source), expected)

    def test_lookalikes_and_other_hosts_are_not_rewritten(self):
        for url in (
            "git://github.com.evil.test/org/repo",
            "git://example.com/github.com/org/repo",
            "http://user@github.com/org/repo",
            "git://github.com:9418/org/repo",
            "https://example.com/github.com/org/repo",
        ):
            with self.subTest(url=url):
                self.assertEqual(normalize_git_url(url), url)

    def test_missing_repository_is_none(self):
        self.assertIsNone(normalize_git_url(None))


if __name__ == "__main__":
    unittest.main()
