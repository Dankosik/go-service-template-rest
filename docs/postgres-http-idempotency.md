<!-- profile:http-idempotency-postgres:start -->
# PostgreSQL HTTP idempotency

`HTTP_IDEMPOTENCY=postgres` retains the one-transaction idempotency component.
It requires `DATABASE=postgres` and remains inert until an OpenAPI operation
declares `x-idempotent: true`, a required `Idempotency-Key` header with
`maxLength: 255`, and the documented 400/401/403/422/500/503/504 responses.

The component scopes a key to the verified caller, operation, and optional
resource; hashes the complete typed request; arbitrates all replicas at the
PostgreSQL writer; and commits the business effect and replayable success in one
transaction. A rollback leaves no reservation. A retry either executes after
that rollback or replays the committed generated response.

Feature code depends only on the neutral executor and a transaction-bound
feature repository:

```go
type handler struct {
	idempotency httpidempotency.Executor[widget.Repository, openapi.CreateWidget201JSONResponse]
}

const createWidgetFingerprintVersion int16 = 1

type createWidgetSemanticInput struct {
	Body widget.CreateInput `json:"body"`
}

func (h *handler) CreateWidget(
	ctx context.Context,
	request openapi.CreateWidgetRequestObject,
) (openapi.CreateWidgetResponseObject, error) {
	principal, ok := reqctx.PrincipalFromContext(ctx)
	if !ok {
		return nil, errors.New("authenticated principal missing")
	}
	caller, ok := principal.CallerIdentity()
	if !ok {
		return nil, errors.New("authenticated principal identity missing")
	}
	idempotencyRequest, err := httpidempotency.NewRequestFromContext(
		ctx,
		httpidempotency.Scope{
			Caller: caller,
			Operation: "createWidget",
		},
		createWidgetFingerprintVersion,
		createWidgetSemanticInput{Body: *request.Body},
	)
	if err != nil {
		return nil, err
	}
	response, _, err := h.idempotency(ctx, idempotencyRequest,
		func(ctx context.Context, widgets widget.Repository) (openapi.CreateWidget201JSONResponse, error) {
			created, err := widgets.Create(ctx, *request.Body)
			return openapi.CreateWidget201JSONResponse{Body: created}, err
		})
	return response, err
}
```

Bootstrap constructs that executor once. `NewExecutor` binds the concrete
repository to the `pgx.Tx`; `JSONCodec` reconstructs the generated success type
without feature-owned response storage:

```go
executor, err := postgresidempotency.NewExecutor(
	idempotency.store,
	func(tx pgx.Tx) widget.Repository { return postgreswidget.New(tx) },
	httpidempotency.JSONCodec[openapi.CreateWidget201JSONResponse](http.StatusCreated),
)
idempotency := executor.Execute
```

The operation-owned semantic input is the fingerprint contract. Keep its
version and JSON representation stable while equivalent requests must replay,
including across rolling deployments and the full retention window. Add fields
only when they change business meaning; normalize transport defaults before
construction. A deliberately incompatible meaning uses a new version, while an
encoding-only change waits for live rows to expire or ships a compatibility
reader first. Never fingerprint the generated request wrapper itself.

Bootstrap appends `httpidempotency.ClassifyError` when the OpenAPI declaration
activates the component, so mismatch, unavailable, and unknown outcomes keep one
stable Problem contract without handler switches.

Authorize every attempt before calling the executor. Include an outbox append
inside the supplied repository transaction when publication is required. An
external API, another datastore, streaming response, or permanent duplicate
guard needs its own design; the component does not pretend those effects join
PostgreSQL.

Deployment supplies only `APP__HTTP_IDEMPOTENCY__RETENTION`, the published
client retry window. Cleanup cadence and batch size are fixed component policy;
each run drains all expired rows through indexed, lock-skipping batches.
Reads and reconciliation use the writer; a replica absence never authorizes
execution. When commit acknowledgement is lost and writer readback cannot
resolve it, return `idempotency_outcome_unknown` and require retry with the same
key.

Migration 9 takes an exclusive table lock and replaces the published reservation
schema only when it is empty. Drain its legacy reservations and replay rows with
the previous release before upgrading; late writers queue behind the migration,
and the migration fails without changing data while any row already exists.
Rollback is likewise refused until the replacement table is empty.
<!-- profile:http-idempotency-postgres:end -->
