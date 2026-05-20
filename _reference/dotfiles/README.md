# Dotfiles: Unified Shell Environment for HPC, Linux, WSL, Codespaces, & Mac

This repo provides an installable, cross-platform dotfiles environment supporting:

* **HPC clusters** (Bash)
* **Linux** (Bash)
* **WSL** (Bash)
* **Devcontainers & GitHub Codespaces** (auto-installs)
* **macOS** (Zsh/Bash; support untested)

It has three purposes:

- Add Git and/or HuggingFace authentication.
- Configure R within VS Code.
- Provide HPC utility scripts for working with `Apptainer` and `SLURM`, and keeping large files within the scratch directory rather than your home directory.

---

## Quick Start

### Local Installation

Clone and install for your environment:

```bash
git clone https://github.com/SATVILab/dotfiles.git "$HOME"/.dotfiles
"$HOME"/.dotfiles/install-env.sh <env>
```

Where `<env>` is one of:

* `hpc` (for clusters)
* `linux` (for normal Linux)
* `wsl` (for Windows Subsystem for Linux)
* `dev` (for generic devcontainer)
* `codespace` (for GitHub Codespaces; installs automatically if set as dotfiles repo)
* `mac` (for Mac; untested)

### GitHub Codespaces

Set this repository as your dotfiles in the Codespaces settings. It will automatically install when you create a new codespace.

---

## What Does This Setup Provide?

* **Utility script installation:**
  All repo scripts go to `~/.local/bin`, including:
  * **Apptainer and Slurm tools** for HPC/cluster users
  * **dotfiles-update** helper for keeping your setup current
* **R config:**
  Copies sensible `.Renviron`, `.lintr`, and `.radian_profile` defaults if using R/VS Code—also strips out HPC-specific R settings if not on cluster.
* **Authentication environment:**
  Prompts (when in interactive mode) to set up initial GitHub and HuggingFace authentication for use in scripts/RStudio/VS Code.
* **Git configuration:**
  Ensures email/username are set and normalises line endings for cross-platform safety.

---

## Updating Your Dotfiles

To update your setup after pulling changes:

```bash
dotfiles-update <env>
```

If in a GitHub codespace, you should rebuild the codespace (full rebuild not needed).

---

## Limitations

* Native Windows is not supported—use WSL.
* macOS support is present but untested.

---

## Forking

To use this as a base for your own dotfiles, fork this repository and modify the scripts as needed.
In particular, the `r/.Renviron` file has SATVI-specific environment variables that you may want to change (harmless if you do not, however).
Feel free to contribute improvements back to the main repository.

---

## Questions or issues?

Feel free to open an issue on the GitHub repository or contact me directly (Miguel Rodo at miguel.rodo@uct.ac.za).

