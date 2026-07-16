# Invoice delegation context

An authenticated support operator may act for a customer user through an approved delegation.
Invoices are tenant-owned and object access must be checked.

Current draft:
- read caller ID from the access token;
- accept X-Subject-ID and X-Tenant-ID from the request;
- allow the read when caller ID equals X-Subject-ID or X-Tenant-ID equals the invoice tenant;
- log the attempted read.

The gateway authenticates the caller but does not authorize delegation, tenant membership, or invoice ownership.
