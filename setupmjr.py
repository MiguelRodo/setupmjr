import subprocess


def run(*args: str, check: bool = True, **kwargs):
    """Run the installed setupmjr CLI."""
    return subprocess.run(("setupmjr", *args), check=check, **kwargs)
