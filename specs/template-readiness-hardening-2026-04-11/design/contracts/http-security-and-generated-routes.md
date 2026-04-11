# HTTP Security And Generated Route Contract

## Public Versus Protected Endpoints

Current baseline endpoints are public system/sample endpoints. The template must not imply that `bearerAuth` is enforced unless a real auth stack exists.

For this hardening task, prefer:

- remove unused `bearerAuth`,
- document that every new business endpoint must declare security intent,
- avoid fake middleware or placeholder tenant behavior.

Future protected endpoint work must define:

- per-operation OpenAPI security,
- 401/403 Problem responses,
- identity extraction middleware,
- tenant/object authorization rules,
- tests that prove protected endpoints reject unauthenticated/unauthorized requests.

## `/metrics`

`/metrics` is operational telemetry, not ordinary business API.

If it stays on the service HTTP server:

- document that production deployments should expose it only on a private scrape path/network or behind real auth,
- keep it as an explicit root-router exception,
- test that no other generated path is shadowed by manual root routing.

If it is removed from OpenAPI:

- keep runtime behavior intentional,
- update OpenAPI runtime contract tests accordingly.

## CORS

Current fail-closed CORS preflight behavior is correct for the baseline. Do not enable browser CORS without a dedicated security decision covering origins, credentials, allowed headers, and protected endpoints.

## Error Details

Generated strict-handler request errors should not echo raw parser error details to clients. Return a stable generic problem detail at the boundary and keep detailed diagnostics in logs with request correlation.

## Manual Route Exceptions

Normal `/api/...` paths belong to generated OpenAPI routing. Manual chi routes are allowed only for documented root-level operational exceptions such as `/metrics`.

