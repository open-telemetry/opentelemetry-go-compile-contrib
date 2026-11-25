// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeclaredMinOtelcVersion(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantFloor string
		wantErr   bool
	}{
		{
			name: "clamps legacy rules up to GLS floor",
			files: map[string]string{
				"a/otelc.yaml":        "version: \"v1.2.0\"\nrules: []\n",
				"b/custom.otelc.yaml": "version: \"v1.0.0\"\n",
			},
			wantFloor: OtelcGLSFloor,
		},
		{
			name: "raises floor when a rule requires newer otelc",
			files: map[string]string{
				"a/otelc.yaml": "version: \"v1.3.0\"\n",
				"b/otelc.yaml": "version: \"v1.4.0\"\n",
			},
			wantFloor: "v1.4.0",
		},
		{
			name: "ignores unversioned legacy file and non-rule yaml",
			files: map[string]string{
				"a/otelc.yaml":   "rules: []\n",
				"b/ignored.yaml": "version: \"v9.9.9\"\n",
			},
			wantFloor: OtelcGLSFloor,
		},
		{
			name: "rejects invalid rule file version",
			files: map[string]string{
				"a/otelc.yaml": "version: \"1.3.0\"\n",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tc.files {
				full := filepath.Join(root, rel)
				require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
				require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
			}

			got, err := DeclaredMinOtelcVersion(root)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantFloor, got)
		})
	}
}

func TestDeclaredMinOtelcVersionRepoRules(t *testing.T) {
	rulesRoot := filepath.Join("..", "..", "instrumentation")
	floor, err := DeclaredMinOtelcVersion(rulesRoot)
	require.NoError(t, err)
	assert.Equal(t, OtelcGLSFloor, floor)
}

func TestResolveOtelcTiers(t *testing.T) {
	const mainPseudo = "v1.2.1-0.20261124161552-e812ed7ef5b2"

	tests := []struct {
		name        string
		minFloor    string
		published   []string
		mainVersion string
		want        []OtelcTier
		wantErr     bool
	}{
		{
			name:        "before v1.3.0 is released only main runs",
			minFloor:    "v1.3.0",
			published:   []string{"v1.0.1", "v1.1.0", "v1.2.0", "v1.2.1", "v1.2.2", "v1.3.0-rc.1"},
			mainVersion: mainPseudo,
			want: []OtelcTier{
				{
					Tier:     "min",
					Runnable: false,
					Reason:   "no published release >= v1.3.0 yet (GLS floor per otelc#1747); covered by main",
				},
				{
					Tier:     "latest",
					Runnable: false,
					Reason:   "latest release v1.2.2 is below the v1.3.0 GLS floor (otelc#1747)",
				},
				{
					Tier:     "main",
					Version:  mainPseudo,
					Runnable: true,
					Reason:   "latest otelc from upstream main",
				},
			},
		},
		{
			name:        "on initial v1.3.0 release latest deduplicates with min",
			minFloor:    "v1.3.0",
			published:   []string{"v1.2.2", "v1.3.0"},
			mainVersion: mainPseudo,
			want: []OtelcTier{
				{
					Tier:     "min",
					Version:  "v1.3.0",
					Runnable: true,
					Reason:   "minimum declared otelc version v1.3.0",
				},
				{
					Tier:     "latest",
					Version:  "v1.3.0",
					Runnable: false,
					Reason:   "matches min tier (v1.3.0); skipping duplicate run",
				},
				{
					Tier:     "main",
					Version:  mainPseudo,
					Runnable: true,
					Reason:   "latest otelc from upstream main",
				},
			},
		},
		{
			name:        "post v1.3.0 with newer releases runs all three tiers",
			minFloor:    "v1.3.0",
			published:   []string{"v1.2.2", "v1.4.0", "v1.3.0", "v1.3.1", "v1.5.0-beta.1"},
			mainVersion: "",
			want: []OtelcTier{
				{
					Tier:     "min",
					Version:  "v1.3.0",
					Runnable: true,
					Reason:   "minimum declared otelc version v1.3.0",
				},
				{
					Tier:     "latest",
					Version:  "v1.4.0",
					Runnable: true,
					Reason:   "latest stable otelc release v1.4.0",
				},
				{
					Tier:     "main",
					Version:  "main",
					Runnable: true,
					Reason:   "latest otelc from upstream main",
				},
			},
		},
		{
			name:     "rejects invalid min floor",
			minFloor: "1.3.0",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveOtelcTiers(tc.minFloor, tc.published, tc.mainVersion)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
