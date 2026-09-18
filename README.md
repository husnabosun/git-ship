# git-ship

`git-ship` automates the whole "get this local folder onto GitHub" chore —
`init` → `.gitignore` → stage → commit → create GitHub repo → sync → push —
in one command, so you never have to remember the exact order again.

Run it from any project directory:

```sh
git-ship
```

## What it does

1. **GitHub account check** — confirms which `gh` account you're authenticated as.
2. **Create GitHub repository** — creates a new repo under that account (skipped if a remote already exists).
3. **git init** — initializes the local repo if it isn't one yet.
4. **git remote add origin** — wires the local repo to the GitHub repo.
5. **Branch prompt** — asks which branch to work on (default `main`).
6. **Checkout branch** — creates/switches to that branch, tracking `origin/<branch>` if it already exists remotely.
7. **git pull (sync with origin)** — pulls any changes made on GitHub first. Uncommitted local edits are auto-stashed before the pull and restored afterward, so a change made on GitHub never gets clobbered by (or blocked by) work you haven't committed yet.
8. **.gitignore** — detects the project's ecosystem (Go, Python, Node, or generic) and writes a matching `.gitignore` if one doesn't exist. Also warns about files that look like secrets (`.env`, keys, etc.) before anything is staged.
9. **git add**
10. **Commit message prompt**
11. **git commit**
12. **git push** — never force-pushes; a rejected (non-fast-forward) push fails loudly instead of being "resolved" automatically.

## Requirements

- [GitHub CLI (`gh`)](https://cli.github.com) installed and authenticated (`gh auth login`).
- Go 1.21+ (only needed to build from source).

## Install / Build

```sh
go build -o git-ship ./cmd/git-ship
```

Then place the resulting binary somewhere on your `PATH`.

## Flags

| Flag         | Description                                                        |
|--------------|---------------------------------------------------------------------|
| `--branch`   | Branch name (skips the interactive prompt).                        |
| `--message`  | Commit message (skips the interactive prompt).                     |
| `--repo`     | GitHub repository name (defaults to the current folder name).      |
| `--public`   | Create the GitHub repository as public (default: private).         |
| `--yes`      | Assume "yes" for non-critical confirmations (never affects push safety). |

## Safety notes

- Push is never forced — if the remote has commits your local branch doesn't, the push fails with a clear message instead of silently overwriting history.
- A merge conflict during the sync-with-origin step aborts the merge and leaves your working tree clean; it never attempts automatic conflict resolution.
