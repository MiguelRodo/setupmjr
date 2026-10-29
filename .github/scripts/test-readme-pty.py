#!/usr/bin/env python3
"""Exercise README line editing on an isolated POSIX pseudo-terminal."""
import fcntl
import os
from pathlib import Path
import pty
import re
import select
import struct
import subprocess
import sys
import tempfile
import termios
import time


class Screen:
    """The VT100 cursor/erase subset emitted by this editor, with normal autowrap."""

    def __init__(self, width):
        self.width = width
        self.rows = [[" "] * width for _ in range(30)]
        self.x = self.y = 0
        self.wrap = False

    def text(self, data):
        data = data.decode("ascii")
        i = 0
        while i < len(data):
            char = data[i]
            if char == "\x1b":
                match = re.match(r"\x1b\[([0-9;]*)([ABCDK])", data[i:])
                assert match, "Unexpected terminal escape sequence"
                amount = int(match[1] or "1")
                command = match[2]
                self.wrap = False
                if command == "A":
                    self.y = max(0, self.y - amount)
                elif command == "B":
                    self.y += amount
                elif command == "C":
                    self.x = min(self.width - 1, self.x + amount)
                elif command == "D":
                    self.x = max(0, self.x - amount)
                else:
                    assert match[1] in ("", "0")
                    self.rows[self.y][self.x:] = [" "] * (self.width - self.x)
                i += len(match[0])
                continue
            if char == "\r":
                self.x, self.wrap = 0, False
            elif char == "\n":
                self.y += 1
                self.wrap = False
            elif char == "\b":
                self.x, self.wrap = max(0, self.x - 1), False
            elif " " <= char <= "~":
                if self.wrap:
                    self.x, self.y, self.wrap = 0, self.y + 1, False
                self.rows[self.y][self.x] = char
                if self.x == self.width - 1:
                    self.wrap = True
                else:
                    self.x += 1
            else:
                assert char == "\a", "Unexpected terminal control character"
            i += 1
        return "".join("".join(row).rstrip() for row in self.rows)


class Session:
    def __init__(self, binary, directory, width):
        self.master, self.slave = pty.openpty()
        fcntl.ioctl(self.slave, termios.TIOCSWINSZ, struct.pack("HHHH", 30, width, 0, 0))
        self.before = termios.tcgetattr(self.slave)
        env = {name: os.environ[name] for name in os.environ if not name.startswith(("OPENAI_", "COPILOT_PROVIDER")) and name != "DEEPSEEK_API_KEY"}
        env["TERM"] = "xterm"
        self.process = subprocess.Popen([binary, "repo", "readme"], cwd=directory, stdin=self.slave, stdout=self.slave, stderr=self.slave, env=env)
        self.output = b""

    def wait_for(self, *tokens):
        deadline = time.monotonic() + 15
        while not any(token in self.output for token in tokens):
            assert time.monotonic() < deadline, "Timed out waiting for README prompt"
            if select.select([self.master], [], [], 0.1)[0]:
                self.output += os.read(self.master, 65536)
            elif self.process.poll() is not None:
                raise AssertionError("README command exited before its next prompt")
        return next(token for token in tokens if token in self.output)

    def send(self, data):
        os.write(self.master, data)

    def finish(self, success=True):
        code = self.process.wait(timeout=15)
        assert (code == 0) == success, "Unexpected README command exit status"
        assert termios.tcgetattr(self.slave) == self.before, "Terminal settings were not restored"

    def close(self):
        if self.process.poll() is None:
            self.process.kill()
            self.process.wait()
        os.close(self.master)
        os.close(self.slave)


def wrapped_backspace(binary, width, erase):
    with tempfile.TemporaryDirectory(prefix="setupmjr-readme-pty-") as directory:
        session = Session(binary, directory, width)
        try:
            session.wait_for(b"Enter title: ")
            session.send(b"Demo\r")
            # Recognize the old prompt so the old binary fails on editing, not wording.
            prompt = session.wait_for(
                b'Complete "This repository ...": ',
                b"Complete this sentence describing the purpose of the repo: This repository... and we'll ensure a single white space after this repository: ",
            )
            prefix = b"0" * (2 * width + (width - 1 - len(prompt) % width) % width)
            session.send(prefix + b"XYZ")
            session.wait_for(b"Z")
            session.send(erase * 3 + b"q")
            session.wait_for(b"q")
            screen = Screen(width).text(session.output)
            erased = not any(char in screen for char in "XYZ")
            displayed_correctly = (prefix + b"q").decode() in screen
            session.send(b"\r")
            for prompts, answer in (
                ((b"Contact 1 name: ", b"Enter Contact 1 Full Name: "), b"Alex Example\r"),
                ((b"Contact 1 email: ", b"Enter Contact 1 Email: "), b"alex@example.test\r"),
                ((b"Contact 2 name (optional): ", b"Enter Contact 2 Full Name: "), b"\r"),
                ((b"Contact 2 email (optional): ", b"Enter Contact 2 Email: "), b"\r"),
            ):
                session.wait_for(*prompts)
                session.send(answer)
            session.finish()
            readme = (Path(directory) / "README.md").read_bytes()
            assert b"This repository " + prefix + b"q\n" in readme
            assert erased, "Saved text was edited correctly, but wrapped backspace left deleted characters visible"
            assert displayed_correctly, "Displayed purpose differs from the edited input"
            assert b"- Alex Example: alex@example.test\n" in readme
            assert readme.endswith(b"\n")
            print(f"PASS: visible wrapped backspace at {width} columns; saved text matches; terminal restored")
        finally:
            session.close()


def cancelled_input(binary, key):
    with tempfile.TemporaryDirectory(prefix="setupmjr-readme-cancel-") as directory:
        session = Session(binary, directory, 40)
        try:
            session.wait_for(b"Enter title: ")
            session.send(key)
            session.finish(success=False)
            assert not (Path(directory) / "README.md").exists()
            print("PASS: cancelled input leaves no README and restores terminal settings")
        finally:
            session.close()


if __name__ == "__main__":
    binary = str(Path(sys.argv[1]).resolve())
    wrapped_backspace(binary, 32, b"\x7f")
    wrapped_backspace(binary, 80, b"\x08")
    cancelled_input(binary, b"\x03")
    cancelled_input(binary, b"\x04")
