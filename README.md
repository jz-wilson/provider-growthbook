# provider-growthbook

A native [Crossplane](https://crossplane.io/) v2 provider for
[GrowthBook](https://www.growthbook.io/), the open-source feature-flag and
experimentation platform. Manage GrowthBook projects, environments, features
(including targeting rules), and SDK connections as Kubernetes resources.

Not built with Upjet: the controllers talk to GrowthBook's REST API directly
through a small typed client, and every resource is covered by unit tests,
live integration tests against a real GrowthBook, and (where the free plan
allows creation) an uptest run in kind.

## Install

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-growthbook
spec:
  package: ghcr.io/jz-wilson/provider-growthbook:v0.1.0
```

Requires Crossplane v2.x. The package declares the `safe-start` capability;
on installations that do not grant CRD watch permission the provider logs
that and starts all controllers ungated instead of failing.

## Configure

Credentials live in a Secret holding a JSON document. `apiUrl` is the API root
including `/api`; omit it for GrowthBook Cloud.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: growthbook-credentials
  namespace: default
stringData:
  credentials: |
    {"apiKey": "secret_...", "apiUrl": "https://growthbook.example.com/api"}
---
apiVersion: growthbook.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: example
  namespace: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: default
      name: growthbook-credentials
      key: credentials
```

A cluster-scoped `ClusterProviderConfig` with the same spec is available too.
See `examples/provider/config.yaml`.

## Resources

| Kind | API group | External name | Notes |
|---|---|---|---|
| `Project` | `core.growthbook.crossplane.io` | GrowthBook id, set after create | name, description, publicId, restrictAccess, stats settings |
| `Environment` | `core.growthbook.crossplane.io` | user-chosen id (defaults to `metadata.name`) | description, toggleOnList, defaultState, projects, parent (create-only) |
| `Feature` | `feature.growthbook.crossplane.io` | user-chosen key (defaults to `metadata.name`) | valueType, defaultValue, description, project, tags, archived, owner, per-environment enabled, rules |
| `SDKConnection` | `sdk.growthbook.crossplane.io` | GrowthBook id, set after create | name, language, environment, projects, payload and proxy options; client key published as connection details |

Every kind above also accepts `spec.initProvider`, mirroring `spec.forProvider`
with every field optional. Values there are applied only when the resource is
first created; `forProvider` wins if a field is set in both, and Crossplane
never treats later drift in an `initProvider`-only field as out of date. Use
it together with a `managementPolicies` value that omits `LateInitialize`.

All managed resources are namespaced (Crossplane v2). Every optional field
is opt-in: fields left unset never cause drift and are late-initialized from
GrowthBook where that makes sense. Examples for each kind live under
`examples/`.

### Feature rules

`spec.forProvider.rules` is an ordered list; when set it is authoritative and
replaces the feature's rules on every update, matching the GrowthBook v2 API.
Supported rule types are `force`, `rollout`, and `experiment-ref`, each with
targeting (`condition`, `savedGroups`) and environment scope
(`allEnvironments` or `environments`). Leave `rules` unset to manage rules
outside Crossplane. Safe-rollout, contextual-bandit, scheduled, and inline
experiment rules, prerequisites, and JSON schema are not modelled yet.

### SDK connection secrets

Set `spec.writeConnectionSecretToRef` on an `SDKConnection` to receive the
client `key`, and when present the `encryptionKey` and `proxySigningKey`.
They are never written to `status`.

### Known API behaviour

- An unlicensed self-hosted GrowthBook enforces the free plan with HTTP 402:
  one project and no custom environments. Features and SDK connections are
  fine.
- GrowthBook may refuse to delete a live feature over REST with 403 until it
  is archived. The provider archives and retries once.
- Some endpoints report a missing object as 400 "Could not find ..." rather
  than 404. The provider treats both as absent.

## Developing

```bash
make submodules       # once, after cloning
make reviewable       # generate, govulncheck, lint, unit tests
make test.integration # against a live GrowthBook, see e2e/README.md
make build            # provider image and xpkg under _output/
```

`e2e/README.md` describes the docker-compose GrowthBook, the bootstrap script
that mints an API key, and the uptest job in `.github/workflows/e2e.yml`.

## Releasing

Push an annotated `v*` tag (or run the Tag workflow). CI builds the package
and publishes `ghcr.io/jz-wilson/provider-growthbook:<tag>`; the `.xpkg`
files are also attached to the GitHub release.

## License

Apache-2.0. Scaffolded from
[crossplane/provider-template](https://github.com/crossplane/provider-template).
