# 40 Data Consistency Cache

## Datastore Config Contract

### Postgres
- `postgres.enabled` controls startup critical probe behavior.
- `postgres.dsn` is mandatory when enabled.
- Only `APP__POSTGRES__*` keys are accepted as env source.

### Redis
- `redis.enabled`, `redis.mode`, `redis.allow_store_mode`, and timeout/pool fields are namespace-only.
- Store mode guard remains explicit (`allow_store_mode=true` required).

### Mongo
- `mongo.enabled` controls optional degraded startup behavior.
- `mongo.uri` and `mongo.database` constraints are namespace-only.

## Consistency Rules
1. No cross-source mutation after snapshot creation.
2. Validation is performed on the final merged snapshot.
3. File source cannot provide secret-like populated values.
4. Env remains the only allowed source for secrets.

## Evolution Policy
- Additive canonical keys only.
- No compatibility translation for non-canonical keys.
