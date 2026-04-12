hey, the http router / handler thing is weird again, specifically `OPTIONS`.
For an existing path I want the normal `Allow` header and `204`, not whatever preflight branch is happening.
But if it's actually CORS preflight and CORS is not enabled, keep that fail closed. I know I'm repeating it, but that's the important split.
There are probably tests nearby, maybe `router_test.go`.
Please don't churn openapi if the public contract does not really change.
Also keep `problem json` stable, don't let that drift.
