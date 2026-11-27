#!/bin/bash

# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

set -EeufCo pipefail

# Open or update a GitHub issue when the VersionMatrix workflow fails.
#
# Dedup strategy: search for open issues labelled `versionmatrix-failure`.
# If one exists, append a comment linking the new failed run.
# If none exists, create a new issue with the label.
#
# Required environment variables:
#   GH_TOKEN  - GitHub token with issues:write permission
#   RUN_URL   - URL of the failing workflow run

: "${GH_TOKEN:?GH_TOKEN must be set}"
: "${RUN_URL:?RUN_URL must be set}"

LABEL="versionmatrix-failure"
TODAY="$(date -u +%Y-%m-%d)"
TITLE="VersionMatrix failed on main (${TODAY})"

existing_issue="$(gh issue list \
  --label "${LABEL}" \
  --state open \
  --limit 1 \
  --json number \
  --jq '.[0].number // empty')"

if [[ -n "${existing_issue}" ]]; then
  echo "Appending comment to existing issue #${existing_issue}"
  gh issue comment "${existing_issue}" \
    --body "VersionMatrix failed again: ${RUN_URL}"
else
  echo "Creating new issue with label '${LABEL}'"
  gh issue create \
    --title "${TITLE}" \
    --label "${LABEL}" \
    --body "$(cat <<EOF
The [VersionMatrix workflow](${RUN_URL}) failed on \`main\`.

This means either:

- A declared library version range in \`instrumentation/.../*.yaml\` does **not** work at one of its bounds (either the instrumented library version cannot be installed because another module forces a different version, or the test suite fails against it), or
- Compatibility broke under one of the tested \`otelc\` compiler tiers (minimum declared floor, latest stable release, or \`@main\`).

## Remediation

1. Identify the failing \`otelc\` tier, test case, and library version from the workflow run.
2. If an instrumented library bound broke, fix the rule's version range in \`instrumentation/.../*.yaml\` (raise the lower bound, cap below the breaking version, or split the rule implementation).
3. If an \`otelc\` compiler tier broke, check whether rule declarations need an updated minimum \`otelc\` \`version:\` or whether an upstream \`otelc\` engine regression needs a fix.
4. Close this issue once the fix lands on \`main\`.

See [docs/testing.md](https://github.com/${GITHUB_REPOSITORY}/blob/main/docs/testing.md#versionmatrix-tests) for full remediation steps.
EOF
)"
fi
