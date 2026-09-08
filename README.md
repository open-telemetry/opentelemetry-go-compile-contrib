<div align="center">
  <img src="https://img.shields.io/badge/Go-1.25%2B-4A90E2?style=flat&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/License-Apache%202.0-4A90E2?style=flat&logo=apache" alt="License" />
  <img src="https://img.shields.io/badge/Status-Experimental-4A90E2?style=flat&logo=github" alt="Status" />
  <img src="https://img.shields.io/badge/Slack-CNCF-FF6B35?style=flat&logo=slack" alt="Slack" />
</div>

# OpenTelemetry Go Compile Contrib

This repository hosts community-contributed instrumentations and supporting packages for
[`opentelemetry-go-compile-instrumentation`][otelc], OpenTelemetry's compile-time instrumentation
tool for Go.

Separating instrumentations from the core `otelc` tool allows new libraries to be instrumented,
reviewed, and released independently of the tool itself, and enables registry-based discoverability
without maintaining temporary manifests in the core repository. See the
[migration epic][migration-epic] for background and status.

[otelc]: https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation
[migration-epic]: https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation/issues/1260

## Status

This repository is newly created and is being populated as part of the migration described in
[open-telemetry/opentelemetry-go-compile-instrumentation#1260][migration-epic]. Instrumentation and
package content will land here in subsequent pull requests.

## Community

### Get Help

- [GitHub Discussions](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation/discussions) - Ask questions
- [GitHub Issues](https://github.com/open-telemetry/opentelemetry-go-compile-contrib/issues) - Report bugs
- [Slack Channel](https://cloud-native.slack.com/archives/C088D8GSSSF) - Real-time chat
- [Calendar](https://github.com/open-telemetry/community/#sig-go-compile-instrumentation) - Community meetings (Thursdays, UTC: 08:00 – 09:00)

## Contributing

We welcome contributions! See our [contributing guide](CONTRIBUTING.md) and [development docs](./docs/).

This project follows the [OpenTelemetry Code of Conduct](https://github.com/open-telemetry/community/blob/main/code-of-conduct.md).
Please also review our [AI usage policy](docs/AI_POLICY.md) if you use AI tools in your workflow.

Here is a list of community roles with current and previous members:

### Maintainers

- [Dario Castañe](https://github.com/darccio), Datadog
- [Haibin Zhang](https://github.com/NameHaibinZhang), Alibaba
- [Huxing Zhang](https://github.com/ralf0131), Alibaba
- [Kemal Akkoyun](https://github.com/kakkoyun), Datadog
- [Przemyslaw Delewski](https://github.com/pdelewski), Quesma
- [Xabier Martinez](https://github.com/txabman42), Cabify
- [Yi Yang](https://github.com/y1yang0), Alibaba

For more information about the maintainer role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#maintainer).

### Approvers

- [Azhar Momin](https://github.com/amazingakai), Independent

For more information about the approver role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#approver).

### Thanks to all of our contributors!

<a href="https://github.com/open-telemetry/opentelemetry-go-compile-contrib/graphs/contributors">
  <img alt="Repo contributors" src="https://contrib.rocks/image?repo=open-telemetry/opentelemetry-go-compile-contrib" />
</a>
