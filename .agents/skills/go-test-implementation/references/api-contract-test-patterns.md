# API Contract Test Patterns

## Behavior Change Thesis
When loaded for HTTP or client-visible API tests, this file makes the model assert the approved transport contract at the handler/generated boundary instead of likely mistake: service-only proof, "any 4xx" assertions, broad end-to-end tests, or guessed status/header/body mappings.

## When To Load
Load this when tests touch HTTP methods, routes, generated OpenAPI handlers, strict request parsing, content type, status codes, response bodies, idempotency headers, request IDs, CORS or fallback behavior, async operation resources, or retry classification.

## Decision Rubric
- Identify the contract source first: OpenAPI, generated handler behavior, existing HTTP tests, approved spec, or explicit bug report.
- Use `httptest.NewRequest` and `httptest.NewRecorder` when handler-level proof observes the contract without a real network.
- Assert method, path, status, content type, required headers, stable response fields, and side-effect suppression only to the exactness approved.
- Keep service/domain tests separate from transport mapping when exact status/header/body semantics are not approved.
- Cover malformed input, unknown fields, trailing JSON, missing required fields, unsupported media type, request size limits, idempotency, retry categories, and request ID behavior when those are part of the changed surface.
- For async start operations, prove `202`/operation-resource semantics and duplicate side-effect suppression only when the contract or current code owns those semantics.
- Do not add manual routes or edit generated files to make a contract test pass unless the approved task ledger owns that change.

## Imitate
```go
func TestCreateWidgetRejectsUnknownJSONField(t *testing.T) {
	handler := NewWidgetHandler(fakeWidgetService{})
	req := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"name":"a","surprise":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q, want problem JSON", got)
	}
}
```

Copy the shape only when strict unknown-field rejection and the status mapping are approved or already established.

```go
func TestCreateWidgetReplaySameIdempotencyKeySuppressesSecondCreate(t *testing.T) {
	service := newRecordingWidgetService()
	handler := NewWidgetHandler(service)

	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"name":"a"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "key-1")
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusAccepted)
		}
	}
	if service.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", service.createCalls)
	}
}
```

Copy the shape only when the API contract owns idempotency-key replay behavior and status mapping.

## Reject
```go
func TestCreateWidgetRejectsUnknownJSONField(t *testing.T) {
	resp := callCreate(`{"name":"a","surprise":true}`)
	if resp.Code >= 400 {
		return
	}
	t.Fatal("request failed")
}
```

Reject because any error status passes and the test cannot catch contract drift between `400`, `413`, `415`, or `422`.

```go
func TestCreateWidgetHTTP(t *testing.T) {
	err := NewWidgetService(fakeStore{}).Create(context.Background(), Widget{Name: "a"})
	if err != nil {
		t.Fatal(err)
	}
}
```

Reject when the obligation is transport behavior. Service proof cannot validate method, headers, decode strictness, or response mapping.

## Agent Traps
- Inventing exact status or problem fields when the approved behavior only says "reject".
- Letting "request failed" mean success for every `>=400` response.
- Starting a real server when handler-level proof would observe the same contract.
- Testing generated/manual route integration by patching generated files.
- Freezing request/correlation ID header names unless the contract or nearby tests already establish them.

## Validation Shape
- Focused handler/contract test command with `-count=1`.
- Package-level HTTP tests when router middleware, generated handler integration, or shared response helpers changed.
- OpenAPI or generated-code drift checks only when the test work changes contract files or generated surfaces.
