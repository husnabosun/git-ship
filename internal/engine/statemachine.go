package engine

import (
	"context"
	"fmt"

	"github.com/husnabosun/git-ship/internal/core"
	"github.com/husnabosun/git-ship/internal/ui"
)

// BuildPipeline returns the full ordered sequence of steps for the "ship"
// flow. Order encodes the dependency graph described in the architecture
// discussion:
//
//  1. GitHub repo must exist before we can add it as a remote.
//  2. Local repo must be initialized before add/commit.
//  3. .gitignore is checked before staging, so untracked secrets are caught.
//  4. Commit message is asked before commit; branch name before branch/push.
//  5. Remote is added only after we know the repo URL and the account.
//  6. Push is always the last step.
//
// Each entry is either a core.Step or a core.PromptStep; RunPipeline
// type-switches between them.
func BuildPipeline() []interface{} {
	return []interface{}{
		AuthInfoStep{},
		GitHubCreateStep{},
		InitStep{},
		RemoteAddStep{},
		BranchNamePromptStep{},
		CheckoutBranchStep{},
		PullStep{},
		GitignoreStep{},
		AddStep{},
		CommitMessagePromptStep{},
		CommitStep{},
		PushStep{},
	}
}

// RunPipeline executes each item in order, printing outcomes as it goes.
// It stops immediately on the first failed Step or the first error from a
// PromptStep (e.g. Ctrl+C / stdin closed).
func RunPipeline(ctx context.Context, s *core.ShipState, pipeline []interface{}) error {
	for _, item := range pipeline {
		switch step := item.(type) {

		case core.PromptStep:
			if step.Skip(s) {
				continue
			}
			if err := step.Prompt(ctx, s); err != nil {
				return fmt.Errorf("%s: %w", step.Name(), err)
			}

		case core.Step:
			if step.CanSkip(s) {
				ui.StepOutcome(step.Name(), core.StepResult{Success: true, Skipped: true, Message: "already satisfied"})
				continue
			}
			result := step.Run(ctx, s)
			ui.StepOutcome(step.Name(), result)
			if !result.Success && !result.Skipped {
				if result.Message != "" {
					ui.Info(result.Message)
				}
				return fmt.Errorf("%s: %w", step.Name(), result.Err)
			}

		default:
			return fmt.Errorf("pipeline item does not implement Step or PromptStep")
		}
	}
	return nil
}
