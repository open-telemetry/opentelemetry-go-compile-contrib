// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

// Package tools tracks tool dependencies used by the Makefile, so `go mod tidy`
// keeps them pinned even though no non-test code imports them directly.
package tools

import (
	_ "github.com/checkmake/checkmake/cmd/checkmake"
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/yamlfmt/cmd/yamlfmt"
	_ "github.com/rhysd/actionlint/cmd/actionlint"
)
