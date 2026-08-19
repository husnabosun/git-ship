package engine

import (
	"context"

	"github.com/husnabosun/git-ship/internal/core"
	"github.com/husnabosun/git-ship/internal/ghclient"
)

// --- GitHubCreateStep -------------------------------------------------------
// Creates a bare repository (no README, no .gitignore, no LICENSE) under the
// authenticated gh account. Runs before RemoteAddStep since it produces the
// remote URL that step needs.

type GitHubCreateStep struct{}

func (GitHubCreateStep) Name() string { return "create GitHub repository" }

func (GitHubCreateStep) CanSkip(s *core.ShipState) bool {
	// If a remote already exists locally, assume the repo already exists
	// remotely too — don't try to create a duplicate.
	return s.HasRemote
}

func (GitHubCreateStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	if s.RepoName == "" {
		s.RepoName = s.FolderName
	}
	remoteURL, err := ghclient.CreateBareRepo(ctx, s.RepoName, s.Visibility)
	if err != nil {
		return core.StepResult{Err: err}
	}
	s.Visibility = "public"
	s.RemoteURL = remoteURL
	s.RepoCreated = true
	return core.StepResult{Success: true, Message: s.RepoName + " (" + s.Visibility + ")"}
}

// --- AuthInfoStep -----------------------------------------------------------
// Purely informational: surfaces which GitHub account will be used, so the
// user notices *before* the push if it's the wrong one — cheaper to catch
// here than after a push to the wrong account's repo.

type AuthInfoStep struct{}

func (AuthInfoStep) Name() string { return "GitHub account check" }

func (AuthInfoStep) CanSkip(s *core.ShipState) bool { return s.GitHubAccount != "" }

func (AuthInfoStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	account, err := ghclient.ActiveAccount(ctx)
	if err != nil {
		return core.StepResult{Err: err}
	}
	s.GitHubAccount = account
	return core.StepResult{Success: true, Message: "authenticated as " + account}
}

// --- RemoteAddStep -----------------------------------------------------------

type RemoteAddStep struct{}

func (RemoteAddStep) Name() string { return "git remote add origin" }

func (RemoteAddStep) CanSkip(s *core.ShipState) bool { return s.HasRemote }

func (RemoteAddStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	if _, err := runGit(ctx, s.RepoPath, "remote", "add", "origin", s.RemoteURL); err != nil {
		return core.StepResult{Err: err}
	}
	s.HasRemote = true
	return core.StepResult{Success: true, Message: s.RemoteURL}
}

// --- PushStep ------------------------------------------------------------------
// Deliberately never force-pushes. A rejected push (non-fast-forward) is
// surfaced as a normal failure with a clear message instead of being
// "resolved" automatically — silently overwriting remote history is the
// single most dangerous thing this tool could do.

type PushStep struct{}

func (PushStep) Name() string { return "git push" }

func (PushStep) CanSkip(s *core.ShipState) bool { return false }

func (PushStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	if _, err := runGit(ctx, s.RepoPath, "push", "-u", "origin", s.BranchName); err != nil {
		return core.StepResult{Err: err, Message: "if this is a non-fast-forward rejection, pull/rebase manually — git-ship will never force-push for you"}
	}
	return core.StepResult{Success: true, Message: "pushed to origin/" + s.BranchName}
}
