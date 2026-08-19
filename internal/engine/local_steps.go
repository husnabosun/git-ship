package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/husnabosun/git-ship/internal/core"
	"github.com/husnabosun/git-ship/internal/preflight"
)

// --- InitStep -----------------------------------------------------------

type InitStep struct{}

func (InitStep) Name() string { return "git init" }

func (InitStep) CanSkip(s *core.ShipState) bool { return s.HasGit }

func (InitStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	if _, err := runGit(ctx, s.RepoPath, "init"); err != nil {
		return core.StepResult{Err: err}
	}
	s.HasGit = true
	return core.StepResult{Success: true}
}

// --- GitignoreStep --------------------------------------------------------
// This step intentionally handles its own y/n prompting rather than being
// split into a separate PromptStep, because the question it asks depends on
// data (missing patterns) that only this step computes. See conversation
// notes: architectural purity is traded for practicality here.

type GitignoreStep struct{}

func (GitignoreStep) Name() string { return ".gitignore check" }

func (GitignoreStep) CanSkip(s *core.ShipState) bool { return false }

func (GitignoreStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	path := filepath.Join(s.RepoPath, ".gitignore")
	content, err := os.ReadFile(path)

	// Branch 1: no .gitignore yet — offer to generate one from a template.
	if err != nil {
		ecosystem := preflight.DetectEcosystem(s.RepoPath)
		if !s.AssumeYes && !askYesNo("No .gitignore found. Create one for detected ecosystem ("+ecosystem+")?", true) {
			return core.StepResult{Success: true, Skipped: true, Message: "user declined"}
		}
		tmpl, loadErr := preflight.LoadTemplate(ecosystem)
		if loadErr != nil {
			return core.StepResult{Err: loadErr}
		}
		if writeErr := os.WriteFile(path, []byte(tmpl), 0644); writeErr != nil {
			return core.StepResult{Err: writeErr}
		}
		s.GitignoreExists = true
		return core.StepResult{Success: true, Message: "generated from " + ecosystem + " template"}
	}

	// Branch 2: .gitignore exists — check for commonly-missed sensitive
	// patterns and ask the user to double-check rather than editing silently.
	s.GitignoreExists = true
	missing := preflight.MissingPatterns(string(content))
	if len(missing) == 0 {
		return core.StepResult{Success: true, Skipped: true, Message: "existing .gitignore looks complete"}
	}

	if s.AssumeYes {
		return core.StepResult{Success: true, Skipped: true, Message: "existing .gitignore kept as-is (--yes)"}
	}

	confirmed := askYesNo(
		".gitignore exists but seems to be missing common patterns ("+joinComma(missing)+"). Did you already double-check it?",
		false,
	)
	if confirmed {
		return core.StepResult{Success: true, Skipped: true, Message: "confirmed by user"}
	}

	if askYesNo("Append the missing patterns now?", true) {
		appendContent := "\n# --- added by git-ship ---\n"
		for _, p := range missing {
			appendContent += p + "\n"
		}
		f, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if openErr != nil {
			return core.StepResult{Err: openErr}
		}
		defer f.Close()
		if _, writeErr := f.WriteString(appendContent); writeErr != nil {
			return core.StepResult{Err: writeErr}
		}
		return core.StepResult{Success: true, Message: "appended " + joinComma(missing)}
	}

	return core.StepResult{Success: true, Skipped: true, Message: "left untouched at user's request"}
}

func joinComma(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ", "
		}
		out += item
	}
	return out
}

// --- AddStep --------------------------------------------------------------

type AddStep struct{}

func (AddStep) Name() string { return "git add ." }

func (AddStep) CanSkip(s *core.ShipState) bool { return false }

func (AddStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	if _, err := runGit(ctx, s.RepoPath, "add", "."); err != nil {
		return core.StepResult{Err: err}
	}
	// Check whether anything was actually staged before asking for a commit
	// message. Asking the user for a message and then failing on commit
	// because there was nothing to commit is a wasted, confusing step.
	out, err := runGit(ctx, s.RepoPath, "diff", "--cached", "--name-only")
	if err != nil {
		return core.StepResult{Err: err}
	}
	if strings.TrimSpace(out) == "" {
		return core.StepResult{Err: fmt.Errorf("no changes to commit — add or edit files in this directory first")}
	}
	return core.StepResult{Success: true}
}

// --- CommitMessagePromptStep ------------------------------------------------

type CommitMessagePromptStep struct{}

func (CommitMessagePromptStep) Name() string { return "commit message" }

func (CommitMessagePromptStep) Skip(s *core.ShipState) bool { return s.CommitMessage != "" }

func (CommitMessagePromptStep) Prompt(ctx context.Context, s *core.ShipState) error {
	for {
		msg := askText("Commit message", "")
		if msg != "" {
			s.CommitMessage = msg
			return nil
		}
		// Commit message is required — re-prompt instead of silently
		// falling back to a generic message a user didn't choose.
	}
}

// --- CommitStep -------------------------------------------------------------

type CommitStep struct{}

func (CommitStep) Name() string { return "git commit" }

func (CommitStep) CanSkip(s *core.ShipState) bool { return false }

func (CommitStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	if _, err := runGit(ctx, s.RepoPath, "commit", "-m", s.CommitMessage); err != nil {
		return core.StepResult{Err: err}
	}
	return core.StepResult{Success: true, Message: s.CommitMessage}
}

// --- BranchNamePromptStep ----------------------------------------------------

type BranchNamePromptStep struct{}

func (BranchNamePromptStep) Name() string { return "branch name" }

func (BranchNamePromptStep) Skip(s *core.ShipState) bool { return s.BranchName != "" }

func (BranchNamePromptStep) Prompt(ctx context.Context, s *core.ShipState) error {
	s.BranchName = askText("Which branch do you want to work on?", "main")
	return nil
}


// --- CheckoutBranchStep ----------------------------------------------------

type CheckoutBranchStep struct{}

func (CheckoutBranchStep) Name() string { return "checkout branch" }

func (CheckoutBranchStep) CanSkip(s *core.ShipState) bool { return false }

func (CheckoutBranchStep) Run(ctx context.Context, s *core.ShipState) core.StepResult {
	dir := s.RepoPath
	branch := s.BranchName

	current, _ := runGit(ctx, dir, "branch", "--show-current")
	if strings.TrimSpace(current) == branch {
		return core.StepResult{Success: true, Skipped: true, Message: "already on " + branch}
	}

	if _, err := runGit(ctx, dir, "rev-parse", "--verify", "refs/heads/"+branch); err == nil {
		if _, err := runGit(ctx, dir, "checkout", branch); err != nil {
			return core.StepResult{Err: err}
		}
		return core.StepResult{Success: true, Message: "switched to existing local branch"}
	}

	if s.HasRemote {
		out, _ := runGit(ctx, dir, "ls-remote", "--heads", "origin", branch)
		if strings.TrimSpace(out) != "" {
			if _, err := runGit(ctx, dir, "checkout", "-b", branch, "origin/"+branch); err != nil {
				return core.StepResult{Err: err}
			}
			return core.StepResult{Success: true, Message: "checked out, tracking origin/" + branch}
		}
	}

	if _, err := runGit(ctx, dir, "checkout", "-b", branch); err != nil {
		return core.StepResult{Err: err}
	}
	return core.StepResult{Success: true, Message: "created new branch"}
}