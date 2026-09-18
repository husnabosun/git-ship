package engine

import (
	"context"
	"strings"

	"github.com/husnabosun/git-ship/internal/core"
)

// PullStep syncs the local branch with origin before any new local work is
// staged or committed. It runs on s.BranchName — the branch the user just
// chose via BranchNamePromptStep/CheckoutBranchStep — not whatever branch
// happened to be checked out before git-ship started.
//
// It only runs when a remote already existed before this run of git-ship.
// If the chosen branch doesn't exist on the remote yet (e.g. a brand new
// branch), that's not an error — there's simply nothing to sync yet.
//
// On a merge conflict, this step deliberately does NOT attempt any
// resolution. It aborts the merge to leave the working tree clean, then
// fails the step so the pipeline stops.
type PullStep struct{}

func (PullStep) Name() string { return "git pull (sync with origin)" }

func (PullStep) CanSkip(s *core.ShipState) bool {
	return !s.HasRemote || s.RepoCreated
}

func (PullStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	// PullStep runs before the local edit gets staged/committed, so any
	// uncommitted work-in-progress must be stashed first — otherwise `git
	// pull` refuses to touch files it would have to overwrite.
	statusOut, _ := runGit(ctx, s.RepoPath, "status", "--porcelain")
	dirty := strings.TrimSpace(statusOut) != ""

	if dirty {
		if _, err := runGit(ctx, s.RepoPath, "stash", "push", "-u", "-m", "git-ship: auto-stash before pull"); err != nil {
			return core.StepResult{Err: err, Message: "failed to stash local changes before pulling"}
		}
	}

	out, err := runGit(ctx, s.RepoPath, "pull", "origin", s.BranchName, "--no-rebase")

	if err == nil {
		if dirty {
			if popOut, popErr := runGit(ctx, s.RepoPath, "stash", "pop"); popErr != nil {
				return core.StepResult{
					Err:     popErr,
					Message: "pulled from origin, but restoring your local changes conflicted: " + strings.TrimSpace(popOut) + " — resolve manually with 'git stash pop', then re-run git-ship.",
				}
			}
		}
		if strings.Contains(out, "Already up to date") {
			return core.StepResult{Success: true, Message: "already up to date"}
		}
		return core.StepResult{Success: true, Message: "synced with origin/" + s.BranchName}
	}

	if strings.Contains(out, "couldn't find remote ref") || strings.Contains(out, "unknown revision or path not in the working tree") {
		if dirty {
			_, _ = runGit(ctx, s.RepoPath, "stash", "pop")
		}
		return core.StepResult{Success: true, Skipped: true, Message: "new branch, nothing to sync yet"}
	}

	if strings.Contains(out, "CONFLICT") || strings.Contains(out, "Automatic merge failed") {
		_, _ = runGit(ctx, s.RepoPath, "merge", "--abort")
		if dirty {
			_, _ = runGit(ctx, s.RepoPath, "stash", "pop")
		}
		return core.StepResult{
			Err:     err,
			Message: "merge conflict with origin/" + s.BranchName + " — merge aborted, working tree left clean. Resolve manually (git pull, fix conflicts, commit) then re-run git-ship.",
		}
	}

	if dirty {
		_, _ = runGit(ctx, s.RepoPath, "stash", "pop")
	}
	return core.StepResult{Err: err}
}