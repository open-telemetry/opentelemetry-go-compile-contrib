# Release Process

`opentelemetry-go-compile-contrib` releases Go modules (`pkg/**`, `instrumentation/**`, and the root module) using [multimod](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/multimod) driven by [`versions.yaml`](../versions.yaml).

## Release Cadence

Releases are cut approximately every **4 weeks** (monthly). Patch releases (hotfixes) are cut as needed between scheduled releases when a critical bug or security issue is found.

## Release Shepherd

Each release is owned by a **release shepherd** — one of the project maintainers. The shepherd is responsible for driving the release end-to-end: preparing the release PR, coordinating review, and publishing the draft GitHub release.

Shepherds rotate among maintainers across releases to share the load.

## Release Schedule

Releases follow a 4-week cadence.

| Version | Target Date | Release Shepherd                                             |
| ------- | ----------- | ------------------------------------------------------------ |
| v0.1.0  | 2026-11-26  | Azhar Momin ([@amazingakai](https://github.com/amazingakai)) |

If you are interested in volunteering to shepherd an upcoming release, please create a pull request proposing yourself for that release.

## Prerequisites

- Maintainer permissions on `open-telemetry/opentelemetry-go-compile-contrib`.
- Authenticated [GitHub CLI (`gh`)](https://cli.github.com/) (`gh auth login`): required by `make release-notes` to call GitHub's `generate-notes` API and fetch PR labels in `.github/tools/releasesummary`.

## How `versions.yaml` Drives Releases

- `module-sets` defines release groups (initially `contrib-all` for lockstep `v0.1.0` releases; additional sets can be added later if new submodules lag behind the top-level group).
- `excluded-modules` lists non-release internal modules (`.tools`, `.github/tools`, `test`, `test/shared/*`, `test/apps/*`). `multimod` requires every `go.mod` in the repository to appear in either `module-sets` or `excluded-modules`. `multimod prerelease` still updates intra-repo `require` lines inside excluded modules without creating git tags for them.
- `release.yml` compares `versions.yaml` against remote git tags to automatically determine which module sets have pending (untagged) versions.

## Cutting a Release

> [!NOTE]
> CI workflows list the maintenance-branch glob `release/v*.*.x` under `on.push.branches` (and `on.pull_request.branches` in `codeql.yml`). GitHub Actions has no shared source for branch filters. If a release-branch naming pattern changes, update every occurrence alongside `release.yml`, or CI will silently stop running on maintenance branches.

### 1. Prepare the Release Branch Locally

1. Create a `release/<version>` branch from `main` (or `release/v*.*.x` for a hotfix):
   ```sh
   git checkout -b release/v0.1.0
   ```
2. Update the target version(s) in `versions.yaml` and commit that change (required because `multimod prerelease` checks for a clean working tree):
   ```sh
   git add versions.yaml
   git commit -m "chore: bump versions.yaml for v0.1.0"
   ```
   _(Skip this commit if `versions.yaml` already has the target version.)_
3. Update all `go.mod` / `go.sum` files and `version.go` references:
   ```sh
   make prerelease
   ```
4. Generate and curate `.github/release-notes.md` (requires authenticated `gh` CLI):
   ```sh
   make release-notes
   ```
5. Review and polish `.github/release-notes.md` if needed, commit the changes, and push the `release/<version>` branch.

### 2. Open and Merge the Release PR

1. Open a pull request from `release/<version>` into `main` titled `release: <version>` (or `release: <ver1>/<ver2>` when releasing multiple distinct set versions).
2. Wait for all required checks (including `check-consistency.yaml`) to pass, obtain maintainer approval, and **merge** the PR.

### 3. Automated Tagging and Draft Release

Merging the `release/...` PR automatically triggers `.github/workflows/release.yml`, which:

1. Validates `versions.yaml` with `multimod verify` and identifies all pending module sets in `versions.yaml`.
2. Computes the prefixed git tags for every module in those pending sets (for example, `v0.1.0`, `pkg/runtime/v0.1.0`, `instrumentation/net/http/otelchttp/v0.1.0`), creates annotated tags on the merge commit, and pushes them to GitHub.
3. Creates a **draft** GitHub release using `.github/release-notes.md` (marked as `--prerelease` if any released version has a pre-release suffix).

### 4. Publish the Draft Release (Manual Gate)

The draft release stays hidden until a maintainer publishes it manually:

1. Open the [Releases page](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/releases).
2. Open the draft release for the new version, review the notes and verify that all module tags were pushed, and click **Publish release**.

## Local Verification

You can verify `versions.yaml` locally at any time:

```sh
make verify-versions
```
