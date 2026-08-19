package engine

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// defaultTimeout guards against a hung `git push` (e.g. dead network,
// waiting on a credential prompt) blocking the CLI forever.
const defaultTimeout = 60 * time.Second

// runGit executes `git <args...>` inside dir and returns combined output.
// Every git invocation in the engine package goes through this function so
// timeout and error wrapping behavior stays consistent.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("git %v timed out after %s (possible network issue or blocked credential prompt)", args, defaultTimeout)
	}
	if err != nil {
		return string(out), fmt.Errorf("git %v failed: %s", args, string(out))
	}
	return string(out), nil
}
