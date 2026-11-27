// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

const (
	// OtelcModulePath is the Go module path of the otelc compiler engine.
	OtelcModulePath = "go.opentelemetry.io/otelc"

	// OtelcGLSFloor is the earliest otelc compiler release that bundles
	// runtime GLS support (opentelemetry-go-compile-instrumentation#1747).
	// Contrib instrumentations have no external instrumentation/runtime package
	// and cannot compile on otelc <= v1.2.x.
	OtelcGLSFloor = "v1.3.0"

	otelcTierMin    = "min"
	otelcTierLatest = "latest"
	otelcTierMain   = "main"
)

// OtelcTier describes one otelc compiler tier evaluated for VersionMatrix.
type OtelcTier struct {
	Tier     string `json:"tier"`
	Version  string `json:"version,omitempty"`
	Runnable bool   `json:"runnable"`
	Reason   string `json:"reason,omitempty"`
}

// DeclaredMinOtelcVersion scans rule YAML files under rulesRoot and returns
// the effective minimum otelc compiler version required to run all rules,
// clamped up to OtelcGLSFloor.
func DeclaredMinOtelcVersion(rulesRoot string) (string, error) {
	effective := OtelcGLSFloor
	err := filepath.WalkDir(rulesRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !isRuleFile(d.Name()) {
			return nil
		}
		data, readErr := os.ReadFile(path) //nolint:gosec // walking rule files under caller-provided rulesRoot
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		var doc struct {
			Version string `yaml:"version"`
		}
		if unmarshalErr := yaml.Unmarshal(data, &doc); unmarshalErr != nil {
			return fmt.Errorf("parse %s: %w", path, unmarshalErr)
		}
		v := strings.TrimSpace(doc.Version)
		if v == "" {
			return nil
		}
		if !semver.IsValid(v) || semver.Prerelease(v) != "" {
			return fmt.Errorf("rule file %s declares invalid otelc release version %q", path, v)
		}
		if semver.Compare(v, effective) > 0 {
			effective = v
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return effective, nil
}

// ResolveOtelcTiers resolves the three VersionMatrix compiler tiers
// ("min", "latest", and "main") against minFloor, the published stable
// release tags for go.opentelemetry.io/otelc, and the current @main version.
func ResolveOtelcTiers(minFloor string, publishedVersions []string, mainVersion string) ([]OtelcTier, error) {
	if !semver.IsValid(minFloor) || semver.Prerelease(minFloor) != "" {
		return nil, fmt.Errorf("invalid minimum otelc floor %q", minFloor)
	}
	stable := stableSemverReleases(publishedVersions)
	minRelease := smallestAtOrAbove(stable, minFloor)

	var newestRelease string
	if len(stable) > 0 {
		newestRelease = stable[len(stable)-1]
	}

	minTier := resolveMinTier(minFloor, minRelease)
	latestTier := resolveLatestTier(minFloor, minRelease, newestRelease)

	mainTarget := strings.TrimSpace(mainVersion)
	if mainTarget == "" {
		mainTarget = otelcTierMain
	}
	mainTier := OtelcTier{
		Tier:     otelcTierMain,
		Version:  mainTarget,
		Runnable: true,
		Reason:   "latest otelc from upstream main",
	}

	return []OtelcTier{minTier, latestTier, mainTier}, nil
}

// QueryOtelcModuleVersions queries the Go toolchain for published release
// versions of go.opentelemetry.io/otelc and the resolved version at @main.
func QueryOtelcModuleVersions(ctx context.Context) ([]string, string, error) {
	versionsOut, err := runGoListJSON(ctx, "list", "-m", "-versions", "-json", OtelcModulePath)
	if err != nil {
		return nil, "", fmt.Errorf("list %s versions: %w", OtelcModulePath, err)
	}
	var modVersions struct {
		Versions []string `json:"Versions"`
	}
	if decodeErr := json.Unmarshal(versionsOut, &modVersions); decodeErr != nil {
		return nil, "", fmt.Errorf("decode %s versions: %w", OtelcModulePath, decodeErr)
	}

	mainOut, err := runGoListJSON(ctx, "list", "-m", "-json", OtelcModulePath+"@main")
	if err != nil {
		return nil, "", fmt.Errorf("resolve %s@main: %w", OtelcModulePath, err)
	}
	var modMain struct {
		Version string `json:"Version"`
	}
	if decodeErr := json.Unmarshal(mainOut, &modMain); decodeErr != nil {
		return nil, "", fmt.Errorf("decode %s@main: %w", OtelcModulePath, decodeErr)
	}
	if strings.TrimSpace(modMain.Version) == "" {
		return nil, "", errors.New("resolved empty version for " + OtelcModulePath + "@main")
	}
	return modVersions.Versions, modMain.Version, nil
}

func resolveMinTier(minFloor, minRelease string) OtelcTier {
	if minRelease == "" {
		return OtelcTier{
			Tier:     otelcTierMin,
			Runnable: false,
			Reason: fmt.Sprintf(
				"no published release >= %s yet (GLS floor per otelc#1747); covered by main",
				minFloor,
			),
		}
	}
	reason := "minimum declared otelc version " + minRelease
	if minRelease != minFloor {
		reason = fmt.Sprintf("earliest published release >= %s floor is %s", minFloor, minRelease)
	}
	return OtelcTier{
		Tier:     otelcTierMin,
		Version:  minRelease,
		Runnable: true,
		Reason:   reason,
	}
}

func resolveLatestTier(minFloor, minRelease, newestRelease string) OtelcTier {
	if newestRelease == "" {
		return OtelcTier{
			Tier:     otelcTierLatest,
			Runnable: false,
			Reason:   "no published stable otelc releases found",
		}
	}
	if semver.Compare(newestRelease, minFloor) < 0 {
		return OtelcTier{
			Tier:     otelcTierLatest,
			Runnable: false,
			Reason: fmt.Sprintf(
				"latest release %s is below the %s GLS floor (otelc#1747)",
				newestRelease,
				minFloor,
			),
		}
	}
	if newestRelease == minRelease {
		return OtelcTier{
			Tier:     otelcTierLatest,
			Version:  newestRelease,
			Runnable: false,
			Reason:   fmt.Sprintf("matches min tier (%s); skipping duplicate run", minRelease),
		}
	}
	return OtelcTier{
		Tier:     otelcTierLatest,
		Version:  newestRelease,
		Runnable: true,
		Reason:   "latest stable otelc release " + newestRelease,
	}
}

func stableSemverReleases(versions []string) []string {
	stable := make([]string, 0, len(versions))
	for _, v := range versions {
		if !semver.IsValid(v) || semver.Prerelease(v) != "" || module.IsPseudoVersion(v) {
			continue
		}
		if !slices.Contains(stable, v) {
			stable = append(stable, v)
		}
	}
	semver.Sort(stable)
	return stable
}

func smallestAtOrAbove(sortedVersions []string, floor string) string {
	for _, v := range sortedVersions {
		if semver.Compare(v, floor) >= 0 {
			return v
		}
	}
	return ""
}

func runGoListJSON(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}
