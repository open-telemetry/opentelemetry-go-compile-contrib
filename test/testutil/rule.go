// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"strings"

	"golang.org/x/mod/semver"
)

// isRuleFile reports whether name is an otelc rule configuration file.
func isRuleFile(name string) bool {
	return name == "otelc.yaml" || name == "otelc.yml" ||
		strings.HasSuffix(name, ".otelc.yaml") || strings.HasSuffix(name, ".otelc.yml")
}

// versionInRange reports whether v falls within the declared version range vr.
// An empty range matches all versions. A single version is a lower bound (inclusive).
// A comma-separated pair "lower,upper" is a half-open interval [lower, upper).
func versionInRange(v, vr string) bool {
	if vr == "" {
		return true
	}
	ensureV := func(s string) string {
		if s != "" && !strings.HasPrefix(s, "v") {
			return "v" + s
		}
		return s
	}
	v = ensureV(v)
	lower, upper, hasUpper := strings.Cut(vr, ",")
	lower = ensureV(lower)
	if semver.Compare(v, lower) < 0 {
		return false
	}
	if !hasUpper {
		return true
	}
	upper = ensureV(upper)
	return semver.Compare(v, upper) < 0
}
