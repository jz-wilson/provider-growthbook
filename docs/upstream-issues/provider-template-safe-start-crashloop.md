# Provider Template CRD Gate Cache Sync Timeout at Startup with safe-start Capability

## Summary

A provider scaffolded from the Crossplane provider-template crashes in a restart loop immediately after installation into Crossplane v2.4.0. The provider pod exits with "timed out waiting for cache to be synced for Kind *v1.CustomResourceDefinition". The root cause is that the installed ProviderRevision reports empty `status.capabilities` even though the package declares `spec.capabilities: [safe-start]`. Crossplane's RBAC manager only authorizes CRD access when safe-start is present in the revision's capabilities, so the provider's service account cannot list CRDs, the informer never syncs, and the startup sequence fails.

## Environment

- Crossplane: v2.4.0 (deployed via Helm chart)
- Cluster: kind
- Provider: generated from crossplane/provider-template HEAD at commit 8bb059f (dated 2026-08-25)
- Packaging: `make build` with default template settings

## Steps to Reproduce

1. Clone crossplane/provider-template at commit 8bb059f.
2. Run `make build` to create a provider package image.
3. Install the provider into Crossplane v2.4.0 with default rbac mode (enabled).
4. Observe the ProviderRevision resource: `kubectl get providerrevision -A -o yaml`.
5. Check the provider pod logs: `kubectl logs -n crossplane-system <provider-pod>`.

## Observed

- Provider pod enters CrashLoopBackOff immediately after installation.
- Pod logs contain: "Cannot start controller manager: failed to wait for crd-gate caches to sync kind source: *v1.CustomResourceDefinition: timed out waiting for cache to be synced for Kind *v1.CustomResourceDefinition" (sometimes reports ProviderConfig cache instead).
- ProviderRevision `status.capabilities` field is empty: `[]`.
- `kubectl auth can-i list customresourcedefinitions --as=system:serviceaccount:crossplane-system:<provider-sa>` returns "no".
- RBAC ClusterRole created by Crossplane for the provider does not include `customresourcedefinitions [get list watch]` permission.

## Expected

- The ProviderRevision should have `status.capabilities: [safe-start]` matching the package's `spec.capabilities` declaration.
- The provider's service account should have permission to list and watch CustomResourceDefinitions.
- The CRD informer should sync successfully and the provider should start without errors.

## Root Cause

The template declares safe-start capability in package/crossplane.yaml but the capability is not propagated to the ProviderRevision status. Crossplane v2.4.0's RBAC manager (internal/controller/rbac/provider/roles/roles.go around line 180) only includes customresourcedefinitions [get list watch] when the ProviderRevision's `status.capabilities` contains "safe-start". Without this RBAC rule, the CRD informer started by customresourcesgate.Setup (cmd/provider/main.go lines 180-182) cannot sync and the provider fails to start.

This is a capability propagation issue in Crossplane's ProviderRevision reconciler, not the template itself. However, the template unconditionally gates startup on CRD access without a fallback, unlike crossplane-contrib/provider-kafka which preflights CRD watch permission and falls back to ungated Setup when it is not available.

## Suggested Fix

Option 1 (preferred): Fix the root cause in Crossplane to ensure ProviderRevision status.capabilities is populated from the package spec.

Option 2 (workaround in template): Add a SelfSubjectAccessReview precheck in cmd/provider/main.go (similar to provider-kafka's canWatchCRD) and conditionally call customresourcesgate.Setup only if the service account has CRD watch permission. If the check fails, log a warning and proceed without the gate.

## References

- crossplane-contrib/provider-kafka issue #102 and commit showing CRD watch precheck
- provider-kafka cmd/provider/main.go canWatchCRD implementation
- Crossplane v2.4.0 internal/controller/rbac/provider/roles/roles.go
- Workaround applied in github.com/jz-wilson/provider-growthbook commit df91c09
