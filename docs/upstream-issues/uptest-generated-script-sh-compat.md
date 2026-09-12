# Uptest Library Mode: Generated Shell Script Incompatible with dash

## Summary

When uptest generates chainsaw steps from library mode templates, the annotate step defines shell functions using bash-specific syntax (arithmetic expansion `((attempt++))` and the `local` keyword) but invokes steps with `/usr/bin/sh -c`, which is dash on Ubuntu 24.04 runners. The generated script fails with "sh: 19: attempt++: not found" and never advances past the first retry attempt. Additionally, the generated template assumes the KUBECTL environment variable is exported but does not default it or document the requirement, causing annotate and other steps to fail if KUBECTL is unset.

## Environment

- uptest: v2.2.0
- GitHub Actions: ubuntu-24.04
- Shell: /usr/bin/sh (dash 0.5.12)
- Kubectl: available in PATH but not exported as KUBECTL env var

## Steps to Reproduce

1. Use uptest v2.2.0 with `uptest e2e <manifests> --use-library-mode`.
2. Run on a GitHub Actions ubuntu-24.04 runner.
3. Trigger a manifest that requires annotation retry (e.g., a resource that becomes Ready after a delay).
4. Observe the generated chainsaw steps in the output or Pod events.
5. Check the step logs for shell errors.

## Observed

- Chainsaw annotate step generates bash-specific retry function in internal/templates/00-apply.yaml.tmpl.
- The function uses `((attempt++))` for arithmetic and `local` for variable scoping.
- When chainsaw runs the step with `/usr/bin/sh -c`, dash does not support these constructs.
- Step logs show: "sh: 19: attempt++: not found" on every retry; the counter never increments and logs always show "Annotation attempt 1/10".
- If KUBECTL environment variable is not exported, the generated command expands to an empty string, resulting in "eval: annotate: not found" or ": command not found" in hack/check_endpoints.sh line 4.
- The apply step times out and fails even though the managed resource is already Ready and Synced.

## Expected

- Generated shell steps should work with POSIX sh (dash) without syntax errors.
- Retry logic should correctly increment the attempt counter.
- The apply step should either default KUBECTL to kubectl or fail clearly with an instruction to export KUBECTL.
- Documentation should specify that KUBECTL must be exported if a non-default kubectl path is needed.

## Root Cause

The template in internal/templates/00-apply.yaml.tmpl assumes a bash execution environment but chainsaw runs script steps with `/usr/bin/sh -c`. Bash extensions like arithmetic expansion and local variables are not portable to dash or POSIX sh. The KUBECTL variable is referenced but never defaulted, leaving it unset when the user does not explicitly export it; bash silently expands an unset variable to empty string (unless set -u is active), but dash may error.

## Suggested Fix

1. Replace bash-specific arithmetic with POSIX arithmetic: change `((attempt++))` to `attempt=$((attempt+1))`.
2. Remove the `local` keyword or explicitly declare variables in POSIX style (functions in POSIX sh do not have a local keyword; variables shadow outer scope by assignment).
3. Alternatively, declare the script step's shell as bash in the chainsaw step definition.
4. Default KUBECTL to kubectl when unset: change `${KUBECTL}` to `${KUBECTL:-kubectl}` in the template and in hack/check_endpoints.sh.
5. Update README or --help output to document that KUBECTL must be exported if a non-default kubectl path is required.

## References

- uptest v2.2.0 internal/templates/00-apply.yaml.tmpl
- uptest hack/check_endpoints.sh line 4
- POSIX sh specification (arithmetic expansion, variable expansion, no local keyword)
- Workaround applied in github.com/jz-wilson/crossplane-provider-growthbook commit 3a252ca (exporting KUBECTL)
- Observed on GitHub Actions ubuntu-24.04, 2026-09-09
