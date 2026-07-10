# Boundary Decoding And Validation

## Behavior Change Thesis

When loaded for boundary-decoding pressure, this file makes the implementation establish one bounded, strict, deterministic input contract before side effects instead of accepting partial payloads, scattering defensive checks, or letting transport details leak into domain logic.

## When To Load

Load this for HTTP or message payload size limits, JSON decoding, unknown or trailing fields, normalization, semantic validation, immutable/read-only inputs, upload metadata, or boundary error classification.

## Decision Rubric

- Apply transport size and media-type limits before expensive parsing or business work.
- Decode exactly one payload. Reject malformed input, unknown fields, and trailing non-whitespace when the approved contract is strict.
- Normalize once at the edge, then validate the normalized representation before side effects.
- Keep syntax, shape, semantic validation, authorization, not-found, conflict, precondition, and internal failures distinct when the approved contract distinguishes them.
- Define omitted, explicit `null`, empty, defaulted, immutable, and read-only field behavior instead of inheriting accidental decoder behavior.
- Keep tenant and actor identity sourced from validated context, not untrusted payload fields or arbitrary headers.
- Bound field-error count and ordering when returning several errors; never disclose stack traces, SQL, secrets, or internal topology.
- Preserve the repository's existing boundary profile unless the approved artifact explicitly changes client-visible behavior.

## Example: Strict JSON Boundary

```go
func decodeCreate(r *http.Request, maxBytes int64, dst *createRequest) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode create request: %w", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode create request: trailing JSON value")
		}
		return fmt.Errorf("decode create request trailer: %w", err)
	}
	return nil
}
```

Adapt the concrete reader and error mapping to the repository. The invariant is bounded input plus one complete payload, not this helper shape.

## Proof Obligations

- oversize input is rejected before store or downstream calls;
- malformed, unknown-field, and trailing-value cases fail deterministically;
- omitted, null, empty, immutable, and read-only behavior matches the approved contract;
- valid input still reaches the owning business path once;
- error output is sanitized and mapped at the transport boundary;
- tests assert both the response and absence of forbidden side effects.

## Traps

- using `io.ReadAll` before applying a size bound;
- decoding one value while silently accepting a second JSON value;
- validating before normalization and then using a different normalized value;
- returning a success status with embedded validation errors;
- widening the public error contract during a local implementation fix;
- adding a generic decoder abstraction whose options hide which boundary policy applies.
