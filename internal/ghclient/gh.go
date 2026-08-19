// Package ghclient shells out to the `gh` CLI. This is the "practical"
// approach agreed on: instead of reimplementing GitHub's REST API and OAuth
// flow, we rely on `gh` (which the user has almost certainly already
// authenticated) for both identity and repo creation.
package ghclient

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const ghTimeout = 30 * time.Second

// ErrGHNotInstalled is returned when the `gh` binary can't be found on PATH.
var ErrGHNotInstalled = errors.New("GitHub CLI ('gh') was not found on PATH — install it from https://cli.github.com and run 'gh auth login'")

// IsInstalled checks whether the gh binary is available.
func IsInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ghTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("gh %v failed: %s", args, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ActiveAccount returns the login name of the currently authenticated gh user.
func ActiveAccount(ctx context.Context) (string, error) {
	if !IsInstalled() {
		return "", ErrGHNotInstalled
	}
	out, err := run(ctx, "api", "user", "--jq", ".login")
	if err != nil {
		return "", fmt.Errorf("could not read active GitHub account, is 'gh auth login' done? (%w)", err)
	}
	return strings.TrimSpace(out), nil
}

// CreateBareRepo creates a new GitHub repository under the authenticated
// account with no README, no .gitignore, and no LICENSE — the project
// directory already has (or will have) those files locally.
// Returns the SSH remote URL to use for `git remote add origin`.
func CreateBareRepo(ctx context.Context, name string, visibility string) (string, error) {
	if !IsInstalled() {
		return "", ErrGHNotInstalled
	}

	visFlag := "--private"
	if visibility == "public" {
		visFlag = "--public"
	}

	// No --add-readme, no --gitignore, no --license: gh only adds those
	// when explicitly asked, so a plain `gh repo create` is already bare.
	if _, err := run(ctx, "repo", "create", name, visFlag); err != nil {
		return "", err
	}

	account, err := ActiveAccount(ctx)
	if err != nil {
		return "", err
	}

	// SSH URL avoids the OS credential-picker prompt that HTTPS remotes
	// can trigger when multiple GitHub accounts are stored on the machine.
	return fmt.Sprintf("git@github.com:%s/%s.git", account, name), nil
}
