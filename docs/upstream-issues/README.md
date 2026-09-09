# Upstream issue drafts

Bug reports drafted against upstream projects while building this provider.
They have not been filed. Review, then open them in the target repository.

| File | Target | Status |
|---|---|---|
| provider-template-safe-start-crashloop.md | crossplane/provider-template | draft; root cause is the build submodule's v1.20.0 CLI pin dropping `capabilities` (fixed here in #1) plus the missing degraded-mode fallback (fixed here in df91c09) |
| uptest-generated-script-sh-compat.md | crossplane/uptest | draft |
