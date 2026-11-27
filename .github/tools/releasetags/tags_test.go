// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeVersionsFile(t *testing.T, content string) string {
	t.Helper()
	vFile := filepath.Join(t.TempDir(), "versions.yaml")
	if err := os.WriteFile(vFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return vFile
}

func TestPlanReleaseSingleSet(t *testing.T) {
	vFile := writeVersionsFile(t, `module-sets:
  contrib-all:
    version: v2.1.0-rc.1
    modules:
      - go.opentelemetry.io/otelc-contrib
      - go.opentelemetry.io/otelc-contrib/pkg/runtime
      - go.opentelemetry.io/otelc-contrib/instrumentation/github.com/ClickHouse/clickhouse-go/v2/otelcclickhouse
`)

	plan, err := planRelease(vFile, map[string]string{}, "", false)
	if err != nil {
		t.Fatalf("planRelease failed: %v", err)
	}
	if plan.BranchSuffix != "v2.1.0-rc.1" || plan.ReleaseTag != "v2.1.0-rc.1" || plan.ReleaseTitle != "v2.1.0-rc.1" {
		t.Errorf("unexpected metadata: %+v", plan)
	}
	if !plan.IsPrerelease || plan.ModuleSetsCSV() != "contrib-all" {
		t.Errorf("unexpected prerelease/sets: %+v", plan)
	}
	wantTags := []string{
		"instrumentation/github.com/ClickHouse/clickhouse-go/v2/otelcclickhouse/v2.1.0-rc.1",
		"pkg/runtime/v2.1.0-rc.1",
		"v2.1.0-rc.1",
	}
	if !reflect.DeepEqual(plan.Tags, wantTags) {
		t.Errorf("tags = %v, want %v", plan.Tags, wantTags)
	}
}

func TestPlanReleaseMultiSetAndSkipping(t *testing.T) {
	vFile := writeVersionsFile(t, `module-sets:
  contrib-all:
    version: v0.5.0
    modules:
      - go.opentelemetry.io/otelc-contrib
      - go.opentelemetry.io/otelc-contrib/pkg/runtime
  experimental-foo:
    version: v0.1.0
    modules:
      - go.opentelemetry.io/otelc-contrib/instrumentation/foo/otelcfoo
  experimental-bar:
    version: v0.2.0
    modules:
      - go.opentelemetry.io/otelc-contrib/instrumentation/bar/otelcbar
`)

	t.Run("root and lagging set pending while bar already tagged", func(t *testing.T) {
		remoteTags := map[string]string{"instrumentation/bar/otelcbar/v0.2.0": "aaaa1111"}
		plan, err := planRelease(vFile, remoteTags, "bbbb2222", false)
		if err != nil {
			t.Fatalf("planRelease failed: %v", err)
		}
		if plan.ModuleSetsCSV() != "contrib-all,experimental-foo" {
			t.Errorf("ModuleSetsCSV = %q, want contrib-all,experimental-foo", plan.ModuleSetsCSV())
		}
		if plan.BranchSuffix != "v0.5.0" || plan.ReleaseTag != "v0.5.0" || plan.ReleaseTitle != "v0.5.0/v0.1.0" {
			t.Errorf("unexpected metadata: %+v", plan)
		}
	})

	t.Run("only non-root set pending", func(t *testing.T) {
		remoteTags := map[string]string{
			"v0.5.0":                              "aaaa1111",
			"pkg/runtime/v0.5.0":                  "aaaa1111",
			"instrumentation/bar/otelcbar/v0.2.0": "aaaa1111",
		}
		plan, err := planRelease(vFile, remoteTags, "bbbb2222", false)
		if err != nil {
			t.Fatalf("planRelease failed: %v", err)
		}
		if plan.BranchSuffix != "experimental-foo/v0.1.0" ||
			plan.ReleaseTag != "instrumentation/foo/otelcfoo/v0.1.0" ||
			plan.ReleaseTitle != "v0.1.0" {
			t.Errorf("unexpected metadata: %+v", plan)
		}
	})

	t.Run("idempotent rerun when all tags already point to HEAD", func(t *testing.T) {
		remoteTags := map[string]string{
			"v0.5.0":                              "aaaa1111",
			"pkg/runtime/v0.5.0":                  "aaaa1111",
			"instrumentation/bar/otelcbar/v0.2.0": "aaaa1111",
			"instrumentation/foo/otelcfoo/v0.1.0": "bbbb2222",
		}
		if _, err := planRelease(vFile, remoteTags, "bbbb2222", false); err == nil {
			t.Fatal("expected error in prepare mode when all tags already exist")
		}
		plan, err := planRelease(vFile, remoteTags, "bbbb2222", true)
		if err != nil || plan.ModuleSetsCSV() != "experimental-foo" {
			t.Fatalf("expected rerun to include experimental-foo, got err=%v plan=%+v", err, plan)
		}
	})

	t.Run("inconsistent tags across commits errors", func(t *testing.T) {
		if _, err := planRelease(vFile, map[string]string{"v0.5.0": "aaaa1111"}, "bbbb2222", true); err == nil {
			t.Fatal("expected error for inconsistent remote tags")
		}
	})
}

func TestLoadRemoteTagsAndWriteGitHubOutput(t *testing.T) {
	dir := t.TempDir()
	lsRemoteFile := filepath.Join(dir, "remote-tags.txt")
	content := strings.Join([]string{
		"1111111111111111111111111111111111111111\trefs/tags/v0.1.0",
		"2222222222222222222222222222222222222222\trefs/tags/v0.1.0^{}",
		"3333333333333333333333333333333333333333\trefs/tags/pkg/runtime/v0.1.0",
	}, "\n")
	if err := os.WriteFile(lsRemoteFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	tags, err := loadRemoteTags(lsRemoteFile)
	if err != nil {
		t.Fatalf("loadRemoteTags failed: %v", err)
	}
	if tags["v0.1.0"] != "2222222222222222222222222222222222222222" {
		t.Errorf("expected peeled SHA for v0.1.0, got %q", tags["v0.1.0"])
	}
	if tags["pkg/runtime/v0.1.0"] != "3333333333333333333333333333333333333333" {
		t.Errorf("expected lightweight SHA for pkg/runtime/v0.1.0, got %q", tags["pkg/runtime/v0.1.0"])
	}

	outFile := filepath.Join(dir, "github-output.txt")
	plan := releasePlan{
		Sets:         []plannedModuleSet{{Name: "contrib-all", Version: "v0.1.0", HasRoot: true}},
		BranchSuffix: "v0.1.0",
		ReleaseTag:   "v0.1.0",
		ReleaseTitle: "v0.1.0",
	}
	if writeErr := writeGitHubOutput(outFile, plan); writeErr != nil {
		t.Fatalf("writeGitHubOutput failed: %v", writeErr)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, wantLine := range []string{
		"branch_suffix=v0.1.0",
		"release_tag=v0.1.0",
		"release_title=v0.1.0",
		"is_prerelease=false",
		"module_sets=contrib-all",
	} {
		if !strings.Contains(string(got), wantLine) {
			t.Errorf("missing %q in github output:\n%s", wantLine, string(got))
		}
	}
}

func TestPlanReleaseAgainstRepo(t *testing.T) {
	versionsFile := filepath.Join("..", "..", "..", "versions.yaml")
	if _, err := os.Stat(versionsFile); err != nil {
		t.Skipf("versions.yaml not found at %s: %v", versionsFile, err)
	}

	plan, err := planRelease(versionsFile, map[string]string{}, "", false)
	if err != nil {
		t.Fatalf("planRelease on repo failed: %v", err)
	}
	if len(plan.Tags) < 35 {
		t.Errorf("expected at least 35 tags in contrib-all, got %d", len(plan.Tags))
	}
}
