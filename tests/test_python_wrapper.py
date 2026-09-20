import unittest
from unittest import mock

import setupmjr


class WrapperTests(unittest.TestCase):
    @mock.patch("setupmjr.subprocess.run")
    def test_run_delegates_to_cli(self, subprocess_run):
        expected = object()
        subprocess_run.return_value = expected

        result = setupmjr.run("git", "profile", capture_output=True)

        self.assertIs(result, expected)
        subprocess_run.assert_called_once_with(
            ("setupmjr", "git", "profile"), check=True, capture_output=True
        )


if __name__ == "__main__":
    unittest.main()
