//go:build ignore

// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package linodego

import (
	"net/url"
	_ "unsafe"
)

//go:linkname setHostURLGetter go.opentelemetry.io/otelc-contrib/instrumentation/github.com/linode/linodego/v2/otelclinodego.setHostURLGetter
func setHostURLGetter(fn func(*Client) string)

func init() {
	setHostURLGetter(func(c *Client) string {
		if c == nil || c.hostURL == "" {
			return ""
		}
		parsed, err := url.Parse(c.hostURL)
		if err != nil {
			return ""
		}
		return parsed.Hostname()
	})
}
