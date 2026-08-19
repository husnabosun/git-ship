// Package core defines the shared data contracts used across every layer
// of git-ship: the state that flows between steps, the result each step
// returns, and the two kinds of steps the state machine can execute.
package core

import "context"

// ShipState is the single source of truth passed between steps.
// No step talks to another step directly — they only read/write this struct.
type ShipState struct {
	// Filesystem context
	RepoPath   string // absolute path of the current working directory
	FolderName string // base name of RepoPath, used as default repo name

	// Local git state (filled by preflight)
	HasGit          bool
	GitignoreExists bool
	CurrentBranch   string // branch already checked out, if repo pre-existed

	// User-provided values (filled by PromptSteps)
	BranchName    string
	CommitMessage string
	RepoName      string // defaults to FolderName, can be overridden
	Visibility    string // "private" or "public"

	// GitHub / remote state (filled by ghclient steps)
	GitHubAccount string // authenticated gh account login
	RemoteURL     string // SSH remote URL, e.g. git@github.com:user/repo.git
	HasRemote     bool
	RepoCreated   bool

	// Flags (non-interactive overrides, set from CLI flags in cmd/main.go)
	AssumeYes bool
}

// StepResult is what every Step returns. UI code decides how to render it —
// the engine and preflight packages never print anything themselves.
type StepResult struct {
	Success bool
	Message string
	Err     error
	Skipped bool
}

// Step is a unit of work that mutates ShipState and/or the filesystem/git repo.
// CanSkip lets the state machine bypass a step without treating it as a failure
// (e.g. skip `git init` if a .git directory already exists).
type Step interface {
	Name() string
	CanSkip(s *ShipState) bool
	Run(ctx context.Context, s *ShipState) StepResult
}

// PromptStep asks the user for input and writes the answer into ShipState.
// It is kept separate from Step because it depends on an interactive TTY,
// which makes it easy to swap out later for flag-based non-interactive input.
type PromptStep interface {
	Name() string
	// Skip lets a prompt be bypassed if the value was already supplied
	// via a CLI flag (e.g. --message "...", --branch main).
	Skip(s *ShipState) bool
	Prompt(ctx context.Context, s *ShipState) error
}
