# Resource Abuse And Cost Controls

## Behavior Change Thesis
When loaded for abuse-prone or expensive behavior, this file makes the model choose principal/tenant-scoped budgets and cheap pre-side-effect gates instead of likely mistake: one global rate limit, reliability-only overload wording, or post-hoc cleanup after expensive work already ran.

## When To Load
Load this when a flow includes repeated attempts, bulk/list/export work, expensive queries, large bodies, provider-cost triggers, background jobs, third-party calls, file processing, batching, concurrency, or user-controlled fan-out.

## Decision Rubric
- Name the scarce resource or abuse target: CPU, DB rows, cache pressure, queue depth, provider cost, email/SMS quota, object storage, memory, goroutines, tenant fairness, or account enumeration.
- Bind limits to the right security principal: caller, subject, tenant, API client, IP or network, object, provider account, idempotency key, or a combination. IP-only is rarely enough for authenticated APIs.
- Put cheap checks before expensive parsing, queries, provider calls, job creation, or side effects. Authentication and tenant binding should happen before principal-scoped budget decisions when the endpoint is protected.
- Define hard limits for body size, page size, batch size, concurrency, retry count, queue depth, fan-out, response size, and job duration when those dimensions are attacker-controlled.
- Define denial semantics: `429` for quota/rate abuse, `413` for body size, `400` or `422` for invalid client-controlled dimensions when that is the repo/API policy, and safe degradation only when it does not leak or perform side effects. For active floods, explicitly decide whether the safer control is to drop or refuse before allocating response work.
- For provider-cost triggers, define attempt budgets, dedup/idempotency, backoff, enumeration resistance, and audit events.

## Imitate
- "Bulk export requires authenticated tenant binding before job creation, caps object count and concurrent jobs per tenant, rejects oversize requests before DB fan-out, and returns `429` without enqueueing work when the tenant budget is exhausted." Copy the principal-scoped budget and pre-queue denial.
- "Password reset and email-send paths budget attempts by subject, caller/client, and coarse network signal, preserve enumeration-resistant responses, and avoid provider calls once the budget is exhausted." Copy the abuse target plus no-provider-cost denial.
- "Search endpoints cap page size, sort fields, and total scanned rows; unapproved dimensions fail validation before query construction." Copy the connection between abuse and query-shape controls.

## Reject
- "Add a rate limit." This is incomplete without principal, resource, budget, window, denial behavior, and proof.
- "Use a global limiter." Global limits can punish unrelated tenants while leaving per-tenant abuse or provider-cost drains intact.
- "Start the job, then cancel if it is too large." The attacker already consumed queue, DB, or provider resources.
- "Reliability owns overload." Reliability owns availability mechanics; security still owns attacker-controlled cost and abuse semantics.

## Agent Traps
- Do not forget authenticated abuse. Valid users and tenants can still drain shared resources or infer data through volume.
- Do not rely on `context` timeout as the only control. Timeouts bound duration but do not provide principal fairness, cost budgets, or enumeration resistance.
- Do not create detailed denial responses that reveal whether an email, account, or object exists unless the API has an explicit disclosure policy.

## Validation Shape
- Budget matrix: abuse target -> controlling principal -> limit/window -> cheap precheck -> denial response -> audit/metric signal -> no-side-effect proof.
- Tests or probes cover oversize body, oversize batch, high concurrency, repeated auth attempts, provider-cost exhaustion, queue/job limits, and allowed traffic immediately after another tenant is limited.
- Assertions include no job enqueued, no provider call, no repository mutation, no unbounded goroutine growth, and no sensitive existence disclosure on denial.
