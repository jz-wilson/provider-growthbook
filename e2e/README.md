# End-to-end testing

Two layers, both driven by a throwaway GrowthBook started from
`docker-compose.yml` in this directory.

## 1. Client integration tests (fast, no cluster)

```bash
docker compose -f e2e/docker-compose.yml up -d --wait
export GROWTHBOOK_API_KEY=$(e2e/bootstrap.sh)
export GROWTHBOOK_API_URL=http://localhost:3100/api
make test.integration
```

`bootstrap.sh` creates the first organization through `/auth/firsttime` (or
logs in if one exists) and mints a `secret_...` API key. The tests are behind
the `integration` build tag and skip when `GROWTHBOOK_API_KEY` is unset.

## 2. uptest against kind (CI only)

`.github/workflows/e2e.yml` builds the provider, installs Crossplane into a
kind cluster, points a `ProviderConfig` at the GrowthBook container, and runs
[uptest](https://github.com/crossplane/uptest) over the manifests in
`e2e/manifests/`. uptest applies each manifest, waits for `Ready`, then
deletes it and waits for the deletion to finish.

## Plan limits on an unlicensed instance

A self-hosted GrowthBook without a `LICENSE_KEY` runs on the free plan. The
API enforces it with HTTP 402:

| Action | Free plan |
|---|---|
| Create a project | 402: setup already created "My First Project", the plan's single one |
| Create a custom environment | 402, only the built-in `production` |
| Update the existing project or `production` environment | allowed |
| Create a feature | allowed — features are not plan-limited |

The integration tests detect the 402 and fall back to update-and-revert on
the resources that exist. The uptest manifests only adopt `production`
with `managementPolicies: ["Observe", "Update", "LateInitialize"]` for the
same reason (Crossplane v2 namespaced resources have no `deletionPolicy`). To exercise create and delete
end to end for project/environment, set `LICENSE_KEY` in `docker-compose.yml`
to a trial or paid key and add create-style manifests under
`e2e/manifests/`.

Features are creatable on the unlicensed free plan, so
`e2e/manifests/feature-basic.yaml` is exercised end to end: uptest creates
it, waits for `Ready`, then deletes it and asserts the deletion completes
(including the provider's archive-then-retry path GrowthBook requires for
deleting a live feature).
