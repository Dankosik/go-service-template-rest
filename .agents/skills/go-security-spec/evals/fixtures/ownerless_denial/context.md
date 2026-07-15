# Export authorization context

POST /admin/exports creates a database row, publishes an export job, and later writes an object-store file.
An existing policy service can answer whether the authenticated principal may export for the tenant.

Current draft:
- unauthorized requests are denied;
- authorization should happen somewhere before completion;
- policy-service errors may be retried;
- add security tests.

The security contract must decide the enforcement owner, behavior when the policy service is unavailable, forbidden side effects on denial, and concrete negative proof. General queue or storage architecture is already approved and is out of scope.
