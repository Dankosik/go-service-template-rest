# Keep template tooling current in derived services

status: ready

Derived services must receive the template's portable development and CI tooling
through the existing `scripts/template-sync.sh` path. A newly published portable
tool, script, validation target, or pinned tool version must not require a manual
copy into every service.

The standard command surface is identical in every derived service. A command
whose capability is absent remains discoverable and returns `not applicable`;
profile initialization no longer deletes the command itself. Service-specific
commands, checks, deployment wiring, secrets, baselines, and provider policy
remain repository-owned and survive adoption unchanged.

The portable surface includes the shared Make implementation, local validation
and generation scripts, pinned Go tools, and lint/generation configuration.
Executable GitHub workflows remain repository-owned because changing their job
topology can change required status contexts. Template-factory checks,
service-specific Gitleaks baselines, CD activation, Railway configuration,
credentials, GitHub rulesets, and deployment environments are not portable.

`template-sync --check` and `--apply` retain their current committed-snapshot,
preflight-before-write, dirty-owned-path refusal, service-owned skill, and
generated-view guarantees. No second sync command or fleet daemon is added.

Legacy services without `template.lock` may adopt portable profile-independent
tooling, but commands whose accepted contract requires the lock continue to fail
closed until a separately reviewed migration records the service's current
identity, profile, and harness choices. The migration never guesses a choice
that current repository evidence cannot prove.

Success means a disposable initialized service can add one local Make target and
one local script, adopt a newer committed template, retain both local owners,
observe the same standard target set and portable file bytes as the template,
resolve the same pinned tools, and pass a repeat zero-drift check. A dirty owned
path or ambiguous legacy profile refuses before writes.

Reopen Specification if portable adoption must merge arbitrary local edits to a
template-owned file, if one standard command must have service-specific meaning,
or if deployment/control-plane state is brought into the portable contract.
