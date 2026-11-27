---
name: Version Release
about: Checklist to follow when shipping a new release.
title: 'Release <version> Checklist'
labels: ''
assignees: ''
---

- [ ] Verify CI is passing on the target branch
- [ ] Prepare the `release/<version>` branch (`versions.yaml`, `make prerelease`, `make release-notes`)
- [ ] Open, review, and approve the [release PR](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/blob/main/docs/RELEASE.md#2-open-and-review-the-release-pr) (verify `LatestLibRun` and `VersionMatrix` pass)
- [ ] Merge the release PR
- [ ] Review pushed module tags and publish the draft release on the [Releases page](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/releases)
- [ ] Verify Go module proxy indexing and [complete post-release steps](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/blob/main/docs/RELEASE.md#post-release)
