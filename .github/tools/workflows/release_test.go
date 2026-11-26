// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package workflows_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflowDoc struct {
	On   yaml.Node            `yaml:"on"`
	Jobs map[string]jobConfig `yaml:"jobs"`
}

type jobConfig struct {
	If          string            `yaml:"if"`
	Permissions map[string]string `yaml:"permissions"`
	Steps       []stepConfig      `yaml:"steps"`
}

type stepConfig struct {
	ID   string         `yaml:"id"`
	Name string         `yaml:"name"`
	If   string         `yaml:"if"`
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
	Run  string         `yaml:"run"`
}

func loadWorkflow(t *testing.T, name string) workflowDoc {
	t.Helper()
	path := filepath.Join("..", "..", "workflows", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var doc workflowDoc
	if unmarshalErr := yaml.Unmarshal(data, &doc); unmarshalErr != nil {
		t.Fatalf("parsing %s: %v", path, unmarshalErr)
	}
	return doc
}

func findReleaseStep(t *testing.T, nameOrID string) stepConfig {
	t.Helper()
	doc := loadWorkflow(t, "release.yml")
	job, ok := doc.Jobs["release"]
	if !ok {
		t.Fatal("release.yml: missing job release")
	}
	for _, step := range job.Steps {
		if step.Name == nameOrID || step.ID == nameOrID {
			return step
		}
	}
	t.Fatalf("release.yml: missing step %q in job release", nameOrID)
	return stepConfig{}
}

func TestReleaseWorkflowExists(t *testing.T) {
	doc := loadWorkflow(t, "release.yml")
	if _, ok := doc.Jobs["release"]; !ok {
		t.Fatal("release.yml: missing job release")
	}
	_ = findReleaseStep(t, "app-token")
	_ = findReleaseStep(t, "plan")
}

func TestAppTokenFailureSurfacesWarning(t *testing.T) {
	step := findReleaseStep(t, "Warn on App token failure")
	if step.If != "steps.app-token.outcome == 'failure'" {
		t.Fatalf("release.yml: warning step must gate on steps.app-token.outcome == 'failure', got %q", step.If)
	}
}

func TestReleaseCheckoutDoesNotPersistCredentials(t *testing.T) {
	step := findReleaseStep(t, "Checkout merge commit")
	if !strings.HasPrefix(step.Uses, "actions/checkout@") {
		t.Fatalf("step uses %q, want actions/checkout", step.Uses)
	}
	if got, ok := step.With["persist-credentials"].(bool); !ok || got {
		t.Fatalf("Checkout persist-credentials = %v (present=%v), want false", step.With["persist-credentials"], ok)
	}
	if got, _ := step.With["token"].(string); got != "${{ steps.app-token.outputs.token || github.token }}" {
		t.Fatalf("Checkout token = %q, want App token with github.token fallback", got)
	}
}

func TestReleaseTagPushUsesExplicitTokenAndServerURL(t *testing.T) {
	planStep := findReleaseStep(t, "Plan tags from versions.yaml")
	if !strings.Contains(planStep.Run, `git ls-remote --tags "${remote_url}"`) {
		t.Error("Plan tags must pass authenticated remote_url to git ls-remote")
	}

	step := findReleaseStep(t, "Create and push tags")
	if !strings.Contains(step.Run, "x-access-token:") {
		t.Error("Create and push tags must authenticate using x-access-token URL")
	}
	if !strings.Contains(step.Run, "${GITHUB_SERVER_URL#https://}") {
		t.Error("Create and push tags must derive host from GITHUB_SERVER_URL")
	}
	if !strings.Contains(step.Run, "${APP_TOKEN:-${GITHUB_TOKEN}}") {
		t.Error("Create and push tags must fall back from APP_TOKEN to GITHUB_TOKEN")
	}
}

func TestReleaseCreatesDraftRelease(t *testing.T) {
	step := findReleaseStep(t, "Create GitHub release")
	if !strings.Contains(step.Run, "--draft") {
		t.Error("Create GitHub release must create a draft release")
	}
}

func TestReleasePRRunsCompatibilityMatrixSuites(t *testing.T) {
	suites := map[string]string{
		"test-latestlibrun.yaml":  "test-latestlibrun",
		"test-versionmatrix.yaml": "test-versionmatrix",
	}
	for file, jobID := range suites {
		t.Run(file, func(t *testing.T) {
			doc := loadWorkflow(t, file)
			job, ok := doc.Jobs[jobID]
			if !ok {
				t.Fatalf("%s: missing job %s", file, jobID)
			}
			if !strings.Contains(job.If, "startsWith(github.head_ref, 'release/')") ||
				!strings.Contains(job.If, "!endsWith(github.head_ref, '.x')") {
				t.Errorf("%s: job %s if=%q must gate PR runs on release/* excluding *.x", file, jobID, job.If)
			}
			notify, ok := doc.Jobs["notify-on-failure"]
			if !ok {
				t.Fatalf("%s: missing notify-on-failure job", file)
			}
			if !strings.Contains(notify.If, "github.event_name == 'schedule'") {
				t.Errorf("%s: notify-on-failure if=%q must be limited to scheduled runs", file, notify.If)
			}
		})
	}
}
