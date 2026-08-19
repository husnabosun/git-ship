package preflight

import (
	"context"
	"os/exec"
	"strings"
)

// GitState is a read-only snapshot of the local repository, gathered before
// any mutating step runs.
type GitState struct {
	HasGit    bool
	Branch    string
	HasRemote bool
	RemoteURL string
}

// InspectGitState checks whether dir is already a git repository and, if so,
// what branch and remote it currently has. It never modifies anything.
func InspectGitState(ctx context.Context, dir string) GitState {
	state := GitState{}

	if !commandSucceeds(ctx, dir, "git", "rev-parse", "--is-inside-work-tree") {
		return state
	}
	state.HasGit = true

	if out, err := runGit(ctx, dir, "branch", "--show-current"); err == nil {
		state.Branch = strings.TrimSpace(out)
	}

	if out, err := runGit(ctx, dir, "remote", "get-url", "origin"); err == nil {
		url := strings.TrimSpace(out)
		if url != "" {
			state.HasRemote = true
			state.RemoteURL = url
		}
	}

	return state
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func commandSucceeds(ctx context.Context, dir string, name string, args ...string) bool {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Run() == nil
}
