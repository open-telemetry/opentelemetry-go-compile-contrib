// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	otelcBinName = "otelc"
	appBinName   = "app"
)

// -----------------------------------------------------------------------------
// E2E Test Infrastructure
// This infrastructure is used to actually build the application with the otelc
// instrumentation tool, execute the application and verify the output.

// newCmd builds an exec.Cmd in dir with the given env. If env is nil, the
// parent process env is used.
func newCmd(ctx context.Context, dir string, env []string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	if env == nil {
		cmd.Env = os.Environ()
	} else {
		cmd.Env = env
	}
	return cmd
}

// repoRoot returns the repository root directory, looking for .tools/go.mod or walking up.
func repoRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := pwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".tools", "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(pwd, "..", ".."), nil
}

// OtelcPath returns the absolute path to the otelc binary.
// It checks:
//  1. os.Getenv("OTELC_PATH")
//  2. .bin/otelc[.exe] in repo root
func OtelcPath() (string, error) {
	if p := os.Getenv("OTELC_PATH"); p != "" {
		return p, nil
	}
	binName := otelcBinName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".bin", binName), nil
}

// appOutputName returns the platform-specific name used for built test binaries.
func appOutputName() string {
	if runtime.GOOS == "windows" {
		return appBinName + ".exe"
	}
	return appBinName
}

// appsPath returns the path to the test apps directory.
func appsPath() (string, error) {
	root, err := repoRoot()
	if err != nil {
		pwd, pwdErr := os.Getwd()
		if pwdErr != nil {
			return "", err
		}
		return filepath.Join(pwd, "..", "apps"), nil
	}
	return filepath.Join(root, "test", "apps"), nil
}

// Build builds the application with the instrumentation tool. The built binary
// is registered for cleanup via t.Cleanup. Standard test apps use
// OTELC_TEST_GOCACHE/<app> as GOCACHE when OTELC_TEST_GOCACHE is set, and drop
// -a so warm builds can reuse that per-app cache. Custom appsDir builds keep
// the caller's arguments and environment unchanged.
func Build(t *testing.T, appsDir, app string, args ...string) {
	t.Helper()
	otelc, err := OtelcPath()
	require.NotEmpty(t, otelc)
	require.NoError(t, err)

	standardAppsDir := appsDir == ""
	var env []string
	if standardAppsDir {
		cacheRoot := os.Getenv("OTELC_TEST_GOCACHE")
		if cacheRoot != "" {
			cacheDir := filepath.Join(cacheRoot, app)
			require.NoError(t, os.MkdirAll(cacheDir, 0o755))
			env = append(os.Environ(), "GOCACHE="+cacheDir)

			// Dropping -a lets warm builds reuse this app-local cache without sharing
			// compiled objects across different instrumentation configs.
			filteredArgs := make([]string, 0, len(args))
			for _, arg := range args {
				if arg != "-a" {
					filteredArgs = append(filteredArgs, arg)
				}
			}
			args = filteredArgs
		}
	}

	output := appOutputName()
	args = append(args, "-o", output)
	args = append([]string{otelc}, args...)

	if standardAppsDir {
		var err error
		appsDir, err = appsPath()
		require.NoError(t, err)
	}
	appDir := filepath.Join(appsDir, app)

	cmd := newCmd(t.Context(), appDir, env, args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(appDir, output))
		_ = os.RemoveAll(filepath.Join(appDir, ".otelc-build"))
		// Left behind only when the otelc subprocess was killed mid-run
		// (release removes it on every normal exit).
		_ = os.Remove(filepath.Join(appDir, ".otelc-build.lock"))
	})
}

// Run runs the application and returns the output. It waits for the
// application to complete. If env is nil, the parent process env is used.
func Run(t *testing.T, appsDir, app string, env []string, args ...string) string {
	t.Helper()
	appName := "./" + appBinName
	if runtime.GOOS == "windows" {
		appName += ".exe"
	}
	if appsDir == "" {
		var err error
		appsDir, err = appsPath()
		require.NoError(t, err)
	}
	appDir := filepath.Join(appsDir, app)
	cmd := newCmd(t.Context(), appDir, env, append([]string{appName}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

// Start starts the application but does not wait for it to complete. If env
// is nil, the parent process env is used. Stdout and stderr are captured and
// logged when the test fails, so that app crashes are visible in CI output.
// The returned AppOutput also lets a test block until the app logs a given
// string, so tests can synchronise on the app reaching a known point.
func Start(t *testing.T, appsDir, app string, env []string, args ...string) (*exec.Cmd, *AppOutput) {
	t.Helper()
	appName := "./" + appBinName
	if runtime.GOOS == "windows" {
		appName += ".exe"
	}
	if appsDir == "" {
		var err error
		appsDir, err = appsPath()
		require.NoError(t, err)
	}
	appDir := filepath.Join(appsDir, app)
	cmd := newCmd(t.Context(), appDir, env, append([]string{appName}, args...)...)

	out := &AppOutput{}
	cmd.Stdout = out
	cmd.Stderr = out

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait() // ensure all output is flushed
		}
		if t.Failed() && out.Len() > 0 {
			t.Logf("app output:\n%s", out)
		}
	})

	return cmd, out
}
