# Release Process

This document describes how to cut a new release of `opentelemetry-go-compile-contrib`.

`opentelemetry-go-compile-contrib` releases Go modules (`pkg/**`, `instrumentation/**`, and the root module) using [multimod](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/multimod) driven by [`versions.yaml`](../versions.yaml).

## Release Cadence

Releases are cut approximately every **4 weeks** (monthly). Patch releases (hotfixes) are cut as needed between scheduled releases when a critical bug or security issue is found.

## Release Shepherd

Each release is owned by a **release shepherd** — one of the project maintainers. The shepherd is responsible for driving the release end-to-end: preparing the release PR, coordinating review, verifying CI and compatibility suites, and publishing the draft GitHub release.

Shepherds rotate among maintainers across releases to share the load.

## Release Schedule

Releases follow a 4-week cadence.

| Version | Target Date | Release Shepherd                                             |
| ------- | ----------- | ------------------------------------------------------------ |
| v0.1.0  | 2026-11-27  | Azhar Momin ([@amazingakai](https://github.com/amazingakai)) |

If you are interested in volunteering to shepherd an upcoming release, please create a pull request proposing yourself for that release.

## Prerequisites

- Maintainer permissions on `open-telemetry/opentelemetry-go-compile-contrib`.
- Authenticated [GitHub CLI (`gh`)](https://cli.github.com/) (`gh auth login`): required by `make release-notes` to call GitHub's `generate-notes` API and fetch PR labels in `.github/tools/releasesummary`.

## Before You Release

1. **Create a [Version Release](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/issues/new?template=version_release.md) checklist issue** to track progress. Fill in the version number in the title.
2. **Ensure the target branch is green.** All required CI workflows must be passing on the commit you intend to release.
3. **Verify there are no pending breaking-change PRs** that should land before this release. Check for PRs labeled `breaking`.
4. **Confirm the version number.** This project follows [Semantic Versioning](https://semver.org/):
   - `PATCH` (`v0.1.Z+1`) — bug fixes, no breaking changes
   - `MINOR` (`v0.X+1.0`) — new features or breaking changes while `v0.x`
   - `MAJOR` (`vX+1.0.0`) — breaking changes once `v1+`

## How `versions.yaml` Drives Releases

- `module-sets` defines release groups (initially `contrib-all` for lockstep `v0.1.0` releases; additional sets such as `experimental-*` can be added later if new submodules lag behind the top-level group).
- `excluded-modules` lists non-release internal modules (`.tools`, `.github/tools`, `test`, `test/shared/*`, `test/apps/*`). `multimod` requires every `go.mod` in the repository to appear in either `module-sets` or `excluded-modules`. `multimod prerelease` still updates intra-repo `require` lines inside excluded modules without creating git tags for them.
- `release.yml` and `make release-notes` compare `versions.yaml` against remote git tags (`git ls-remote --tags`) to automatically determine which module sets have pending (untagged) versions.

## Release Steps

> [!NOTE]
> CI workflows list the maintenance-branch glob `release/v*.*.x` under `on.push.branches` (and `on.pull_request.branches` in `codeql.yml`). GitHub Actions has no shared source for branch filters. If a release-branch naming pattern changes, update every occurrence alongside `release.yml`, or CI will silently stop running on maintenance branches.

### 1. Prepare the Release Branch Locally

1. Create a `release/<version>` branch from `main` (or from `release/v<MAJOR>.<MINOR>.x` for a hotfix):

   ```sh
   git checkout -b release/v0.1.0
   ```

2. Update the target version(s) in `versions.yaml` and commit that change (required because `multimod prerelease` checks for a clean working tree):

   ```sh
   git add versions.yaml
   git commit -m "chore: bump versions.yaml for v0.1.0"
   ```

   *(Skip this commit if `versions.yaml` already has the target version.)*
3. Update all `go.mod` / `go.sum` files and `version.go` references across the repository:

   ```sh
   make prerelease
   ```

   *(This runs `multimod verify`, `multimod prerelease --skip-go-mod-tidy --commit-to-different-branch=false`, and `make go-mod-tidy` across all modules.)*
4. Generate and curate `.github/release-notes.md` (requires authenticated `gh` CLI):

   ```sh
   make release-notes
   ```

   The generated file contains a curated summary (`Breaking Changes`, `Added`, `Fixed`, `Changed`) above GitHub's complete `What's Changed` list. Routine maintenance, test, and dependency PRs remain listed under `What's Changed`.
5. Review and edit `.github/release-notes.md` to add context, reword entries, or highlight important changes, then commit and push the `release/<version>` branch.

### 2. Open and Review the Release PR

1. Open a pull request from `release/<version>` into `main` (or `release/v<MAJOR>.<MINOR>.x` for a hotfix) titled `release: <version>` (or `release: <ver1>/<ver2>` when releasing multiple distinct set versions).
2. `release/*` PRs (excluding `*.x` branches) automatically run the `LatestLibRun` and `VersionMatrix` compatibility suites in addition to standard CI checks.
3. Wait for all checks to pass, obtain maintainer approvals, and merge the PR.

### 3. Automated Tagging and Draft Release

Merging the approved `release/...` PR automatically triggers [`.github/workflows/release.yml`](../.github/workflows/release.yml), which:

1. Validates `versions.yaml` with `make verify-versions` and identifies all pending module sets in `versions.yaml`.
2. Computes the prefixed git tags for every module in those pending sets (for example, `v0.1.0`, `pkg/runtime/v0.1.0`, `instrumentation/net/http/otelchttp/v0.1.0`), creates annotated tags on the merge commit, and pushes them to GitHub.
3. Creates the GitHub release as a **draft** using `.github/release-notes.md` (and marks it `--prerelease` if any released version has a pre-release suffix).

The draft release stays invisible on the Releases page until a maintainer publishes it manually. The git tags already exist at this point; publishing the draft is the final gate before the release notes become public.

### 4. Publish the Draft Release (Manual Gate)

1. Open the [Releases page](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/releases).
2. Open the draft release for the new version, review the notes and verify that all module tags were pushed, and click **Publish release**.

## Pre-Releases

To cut a release candidate (e.g., `v0.2.0-rc.1`):

1. Set the pre-release version (e.g., `v0.2.0-rc.1`) in `versions.yaml` on branch `release/v0.2.0-rc.1` and follow the standard release steps.
2. `release.yml` automatically marks the draft GitHub release with `--prerelease`.
3. When cutting the final release (`v0.2.0`) later, `make release-notes` ignores `-rc` tags when finding `previous_tag_name` so the final release notes cover the full release cycle.

## Patch Releases (Hotfixes)

No permanent release or maintenance branches are kept. Critical bug fixes should always land on `main` first. For patch releases between scheduled releases:

1. **If no unrelated changes have landed on `main`** since the last release, create `release/v<MAJOR>.<MINOR>.<PATCH>` directly from `main` and follow the standard release steps.
2. **If `main` has moved on** with changes that should not be included, create a temporary hotfix branch `release/v<MAJOR>.<MINOR>.x` from the previous release tag and cherry-pick the fix commit from `main`:

   ```sh
   git fetch origin --tags
   git checkout -b release/v0.1.x v0.1.0
   git cherry-pick <fix-commit-from-main>
   git push origin release/v0.1.x
   ```

   Then prepare a `release/v0.1.1` branch from `release/v0.1.x`, open the `release: v0.1.1` PR targeting `release/v0.1.x`, review and merge it, publish the draft release, and delete the temporary `release/v0.1.x` branch once verified.

## Post-Release

After publishing the draft release:

1. **Verify Go module proxy indexing** for the published modules:

   ```sh
   GOPROXY=https://proxy.golang.org go list -m go.opentelemetry.io/otelc-contrib/instrumentation/net/http/otelchttp@v0.1.0
   ```

2. **Announce the release** in the [#otel-go-compt-instr-sig](https://cloud-native.slack.com/archives/C088D8GSSSF) Slack channel with a link to the release notes.
3. **Clean up temporary branches** (`release/<version>` and, if used, `release/v<MAJOR>.<MINOR>.x`).
4. **Close the release checklist issue.**

## Breaking Changes

Breaking changes are detected automatically. When a PR title uses the conventional commit `!` modifier (e.g., `feat!: remove deprecated API` or `fix(scope)!: change default behavior`), the `breaking` label is added automatically by CI. This label controls categorization under **Breaking Changes** in `.github/release-notes.md`.

## Local Verification

You can verify `versions.yaml` locally at any time:

```sh
make verify-versions
```
