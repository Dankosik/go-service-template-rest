# Checkout ownership

- The HTTP handler owns decoding, transport validation, and response mapping only.
- `CheckoutService` owns checkout orchestration, including repository access, payment, cache updates, and effect ordering.
- The handler must delegate one application request to `CheckoutService`; it must not construct a second orchestration path.
- Retry or fallback policy requires an accepted specification decision before implementation.
