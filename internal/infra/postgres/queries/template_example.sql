-- TEMPLATE EXAMPLE: delete this query if SQLC transactions are not needed, or
-- replace it with feature-owned SQL and remove the marker.
-- name: TemplateExampleTransactionID :one
SELECT pg_current_xact_id()::text AS transaction_id;
