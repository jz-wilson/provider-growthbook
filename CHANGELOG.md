# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [v0.1.0] - 2026-09-09

First release. Crossplane v2 provider for GrowthBook with four namespaced
managed resources.

### Added

- `Project` (`core.growthbook.crossplane.io`): name, description, publicId,
  restrictAccess, statistics settings. GrowthBook-assigned id as external name.
- `Environment` (`core.growthbook.crossplane.io`): description, toggleOnList,
  defaultState, projects, create-only parent. User-chosen id as external name.
- `Feature` (`feature.growthbook.crossplane.io`): valueType, defaultValue,
  description, project, tags, archived, owner, per-environment enabled
  toggles, and rules (`force`, `rollout`, `experiment-ref`) with targeting
  and environment scope. Archives before delete when the API requires it.
- `SDKConnection` (`sdk.growthbook.crossplane.io`): name, language,
  environment, projects, payload and proxy options. Client key, encryption key
  and proxy signing key are published only as connection details.
- `ProviderConfig` and `ClusterProviderConfig` reading a JSON credentials
  Secret (`apiKey`, optional `apiUrl`).
- Degraded-mode startup: when the service account cannot watch CRDs the
  provider starts all controllers without safe-start instead of crash-looping.
- Unit tests for every controller and the client; live integration tests
  behind the `integration` build tag; uptest e2e in kind for Environment and
  Feature; govulncheck in `make reviewable`.

### Known limitations

- Feature prerequisites, JSON schema, scheduled rules, safe-rollout,
  contextual-bandit, and inline experiment rules are not modelled.
- An unlicensed self-hosted GrowthBook allows one project and no custom
  environments (HTTP 402); the provider surfaces the error as-is.

[v0.1.0]: https://github.com/jz-wilson/provider-growthbook/releases/tag/v0.1.0
