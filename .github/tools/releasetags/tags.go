// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	rootModulePath    = "go.opentelemetry.io/otelc-contrib"
	lsRemoteMinFields = 2
)

var semverRegex = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$`)

type versionsConfig struct {
	ModuleSets map[string]moduleSetConfig `yaml:"module-sets"`
}

type moduleSetConfig struct {
	Version string   `yaml:"version"`
	Modules []string `yaml:"modules"`
}

type plannedModuleSet struct {
	Name    string
	Version string
	HasRoot bool
	Tags    []string
}

type releasePlan struct {
	Sets         []plannedModuleSet
	BranchSuffix string
	ReleaseTag   string
	ReleaseTitle string
	IsPrerelease bool
	Tags         []string
}

func (p releasePlan) ModuleSetsCSV() string {
	names := make([]string, len(p.Sets))
	for i, s := range p.Sets {
		names[i] = s.Name
	}
	return strings.Join(names, ",")
}

// loadRemoteTags parses `git ls-remote --tags` output into tag -> commit SHA,
// preferring peeled annotated entries (refs/tags/<tag>^{}).
func loadRemoteTags(path string) (map[string]string, error) {
	tags := make(map[string]string)
	if path == "" {
		return tags, nil
	}

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	peeled := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < lsRemoteMinFields {
			continue
		}
		sha, ref := fields[0], fields[1]
		tagRef, ok := strings.CutPrefix(ref, "refs/tags/")
		if !ok {
			continue
		}
		if tagName, isPeeled := strings.CutSuffix(tagRef, "^{}"); isPeeled {
			tags[tagName] = sha
			peeled[tagName] = true
		} else if !peeled[tagRef] {
			tags[tagRef] = sha
		}
	}
	return tags, scanner.Err()
}

// planRelease inspects versions.yaml against remoteTags and returns pending
// module sets and derived release metadata.
func planRelease(
	versionsFile string,
	remoteTags map[string]string,
	headCommit string,
	allowAllAtHead bool,
) (releasePlan, error) {
	data, err := os.ReadFile(filepath.Clean(versionsFile))
	if err != nil {
		return releasePlan{}, fmt.Errorf("reading %s: %w", versionsFile, err)
	}
	var cfg versionsConfig
	if unmarshalErr := yaml.Unmarshal(data, &cfg); unmarshalErr != nil {
		return releasePlan{}, fmt.Errorf("parsing %s: %w", versionsFile, unmarshalErr)
	}

	setNames := make([]string, 0, len(cfg.ModuleSets))
	for name := range cfg.ModuleSets {
		setNames = append(setNames, name)
	}
	sort.Strings(setNames)

	var pending []plannedModuleSet
	for _, name := range setNames {
		modSet := cfg.ModuleSets[name]
		if !semverRegex.MatchString(modSet.Version) {
			return releasePlan{}, fmt.Errorf("module set %q has invalid version %q", name, modSet.Version)
		}
		if len(modSet.Modules) == 0 {
			return releasePlan{}, fmt.Errorf("module set %q has no modules", name)
		}

		tags, hasRoot, tagErr := moduleSetTags(name, modSet)
		if tagErr != nil {
			return releasePlan{}, tagErr
		}
		include, stateErr := classifyTagState(
			name+" ("+modSet.Version+")", tags, remoteTags, headCommit, allowAllAtHead,
		)
		if stateErr != nil {
			return releasePlan{}, stateErr
		}
		if include {
			pending = append(pending, plannedModuleSet{
				Name: name, Version: modSet.Version, HasRoot: hasRoot, Tags: tags,
			})
		}
	}

	if len(pending) == 0 {
		return releasePlan{}, errors.New("no unreleased module-sets found in versions.yaml")
	}
	return buildReleasePlan(pending), nil
}

func moduleSetTags(name string, modSet moduleSetConfig) ([]string, bool, error) {
	tags := make([]string, 0, len(modSet.Modules))
	hasRoot := false
	for _, modPath := range modSet.Modules {
		if modPath == rootModulePath {
			hasRoot = true
			tags = append(tags, modSet.Version)
			continue
		}
		rel, ok := strings.CutPrefix(modPath, rootModulePath+"/")
		if !ok {
			return nil, false, fmt.Errorf("module %q in %q must start with %s/", modPath, name, rootModulePath)
		}
		tags = append(tags, rel+"/"+modSet.Version)
	}
	sort.Strings(tags)
	return tags, hasRoot, nil
}

func classifyTagState(
	label string,
	tags []string,
	remoteTags map[string]string,
	headCommit string,
	allowAllAtHead bool,
) (bool, error) {
	atHead, atOther, missing := 0, 0, 0
	for _, tag := range tags {
		sha, exists := remoteTags[tag]
		switch {
		case !exists:
			missing++
		case headCommit != "" && sha == headCommit:
			atHead++
		default:
			atOther++
		}
	}
	if atOther == len(tags) && missing == 0 && atHead == 0 {
		return false, nil
	}
	if atOther > 0 {
		return false, fmt.Errorf(
			"module set %s has inconsistent remote tags: %d on other commits, %d on HEAD, %d missing",
			label, atOther, atHead, missing,
		)
	}
	if missing == 0 {
		return allowAllAtHead && atHead == len(tags), nil
	}
	return true, nil
}

func buildReleasePlan(pending []plannedModuleSet) releasePlan {
	sort.SliceStable(pending, func(i, j int) bool {
		if pending[i].HasRoot != pending[j].HasRoot {
			return pending[i].HasRoot
		}
		return pending[i].Name < pending[j].Name
	})

	primary := pending[0]
	branchSuffix, releaseTag := primary.Version, primary.Version
	if !primary.HasRoot {
		branchSuffix = primary.Name + "/" + primary.Version
		releaseTag = primary.Tags[0]
	}

	seen := make(map[string]bool)
	var versions, allTags []string
	isPrerelease := false
	for _, s := range pending {
		if !seen[s.Version] {
			seen[s.Version] = true
			versions = append(versions, s.Version)
		}
		if strings.Contains(s.Version, "-") {
			isPrerelease = true
		}
		allTags = append(allTags, s.Tags...)
	}
	sort.Strings(allTags)

	return releasePlan{
		Sets:         pending,
		BranchSuffix: branchSuffix,
		ReleaseTag:   releaseTag,
		ReleaseTitle: strings.Join(versions, "/"),
		IsPrerelease: isPrerelease,
		Tags:         allTags,
	}
}

func writeGitHubOutput(path string, plan releasePlan) error {
	var b strings.Builder
	fmt.Fprintf(&b, "branch_suffix=%s\n", plan.BranchSuffix)
	fmt.Fprintf(&b, "release_tag=%s\n", plan.ReleaseTag)
	fmt.Fprintf(&b, "release_title=%s\n", plan.ReleaseTitle)
	fmt.Fprintf(&b, "is_prerelease=%s\n", strconv.FormatBool(plan.IsPrerelease))
	fmt.Fprintf(&b, "module_sets=%s\n", plan.ModuleSetsCSV())

	f, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if _, writeErr := f.WriteString(b.String()); writeErr != nil {
		_ = f.Close()
		return writeErr
	}
	return f.Close()
}
