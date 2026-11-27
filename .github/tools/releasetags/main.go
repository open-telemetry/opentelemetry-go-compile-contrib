// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This tool reads versions.yaml, finds module sets whose versions are not yet
// tagged on remote, writes GitHub Actions outputs, and prints module git tags
// so CI can create annotated tags without requiring a GPG signing key.
//
// TODO: Making GPG signing optional in `multimod tag` should be made upstream
// (https://github.com/open-telemetry/opentelemetry-go-build-tools/blob/main/multimod/internal/tag/tag.go#L203
// hardcodes `-s`, which cannot run on CI with GitHub Apps); for now we have
// releasetags to do this.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(os.Args[1:], logger); err != nil {
		logger.Error("releasetags failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string, logger *slog.Logger) error {
	var (
		versionsFile, remoteTagsFile string
		headCommit, githubOutput     string
		allowAllAtHead, printTags    bool
	)

	flags := flag.NewFlagSet("releasetags", flag.ContinueOnError)
	flags.StringVar(&versionsFile, "versions-file", "versions.yaml", "Path to versions.yaml")
	flags.StringVar(&remoteTagsFile, "remote-tags-file", "", "Path to git ls-remote --tags output")
	flags.StringVar(&headCommit, "head-commit", "", "Current HEAD commit SHA for idempotent tag checks")
	flags.StringVar(&githubOutput, "github-output", "", "Path to GITHUB_OUTPUT file")
	flags.BoolVar(&allowAllAtHead, "allow-all-at-head", false, "Include sets whose tags already point to HEAD")
	flags.BoolVar(&printTags, "print-tags", false, "Print git tag names for pending module sets")

	if err := flags.Parse(args); err != nil {
		return err
	}

	remoteTags, err := loadRemoteTags(remoteTagsFile)
	if err != nil {
		return fmt.Errorf("loading remote tags: %w", err)
	}

	plan, err := planRelease(versionsFile, remoteTags, headCommit, allowAllAtHead)
	if err != nil {
		return fmt.Errorf("planning release: %w", err)
	}

	if githubOutput != "" {
		if writeErr := writeGitHubOutput(githubOutput, plan); writeErr != nil {
			return fmt.Errorf("writing github output: %w", writeErr)
		}
	}

	if !printTags {
		logger.Info(
			"planned release",
			"module_sets", plan.ModuleSetsCSV(),
			"release_title", plan.ReleaseTitle,
			"tag_count", len(plan.Tags),
		)
		return nil
	}

	for _, tag := range plan.Tags {
		if _, writeErr := fmt.Fprintln(os.Stdout, tag); writeErr != nil {
			return fmt.Errorf("writing tag output: %w", writeErr)
		}
	}
	return nil
}
