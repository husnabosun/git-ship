// Command git-ship automates the init → gitignore → commit → GitHub repo →
// push sequence for a fresh project directory, so you never have to
// remember the exact order of commands again.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/husnabosun/git-ship/internal/core"
	"github.com/husnabosun/git-ship/internal/engine"
	"github.com/husnabosun/git-ship/internal/ghclient"
	"github.com/husnabosun/git-ship/internal/preflight"
	"github.com/husnabosun/git-ship/internal/ui"
)

func main() {
	var (
		branchFlag  = flag.String("branch", "", "branch name (skips the interactive prompt)")
		messageFlag = flag.String("message", "", "commit message (skips the interactive prompt)")
		repoFlag    = flag.String("repo", "", "GitHub repository name (defaults to the folder name)")
		publicFlag  = flag.Bool("public", false, "create the GitHub repository as public (default: private)")
		yesFlag     = flag.Bool("yes", false, "assume 'yes' for non-critical confirmations (does not affect push safety)")
	)
	flag.Parse()

	ctx := context.Background()

	if !ghclient.IsInstalled() {
		ui.Fail(ghclient.ErrGHNotInstalled.Error())
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		ui.Fail("could not determine current directory: " + err.Error())
		os.Exit(1)
	}

	visibility := "private"
	if *publicFlag {
		visibility = "public"
	}

	state := &core.ShipState{
		RepoPath:      cwd,
		FolderName:    filepath.Base(cwd),
		RepoName:      *repoFlag,
		BranchName:    *branchFlag,
		CommitMessage: *messageFlag,
		Visibility:    visibility,
		AssumeYes:     *yesFlag,
	}

	// Preflight: inspect existing git state without mutating anything.
	gitState := preflight.InspectGitState(ctx, state.RepoPath)
	state.HasGit = gitState.HasGit
	state.HasRemote = gitState.HasRemote
	state.RemoteURL = gitState.RemoteURL
	state.CurrentBranch = gitState.Branch
	if gitState.Branch != "" {
		ui.Info(fmt.Sprintf("Existing repository detected on branch '%s'.", gitState.Branch))
	}

	// Preflight: warn about likely secrets before anything gets staged.
	if suspicious := preflight.ScanForSensitiveFiles(state.RepoPath); len(suspicious) > 0 {
		ui.Warn("Found files that look sensitive — make sure they end up in .gitignore:")
		for _, f := range suspicious {
			fmt.Println("    - " + f)
		}
	}

	pipeline := engine.BuildPipeline()
	if err := engine.RunPipeline(ctx, state, pipeline); err != nil {
		ui.Fail(err.Error())
		os.Exit(1)
	}

	ui.Info("Done. Repository is live at: https://github.com/" + state.GitHubAccount + "/" + state.RepoName)
}
