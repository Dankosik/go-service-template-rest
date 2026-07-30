# Reference Selector

State which architecture decision the selected reference can change.

| Pressure | Load | Required effect |
| --- | --- | --- |
| Boundary/write ownership, shared data, team seams, or package layout is confused with service architecture | [boundary-decomposition-examples.md](boundary-decomposition-examples.md) | Choose invariant and authority boundaries, leaving package placement downstream. |
| New service-to-service call, internal API, gRPC vs REST/OpenAPI, consumer-class change, or protocol migration | [inter-service-protocol-selection.md](inter-service-protocol-selection.md) | Classify the consumers and choose one canonical protocol; default a new strictly internal synchronous call to gRPC when no accepted requirement or current constraint defeats it. |
| Modular monolith, separate runtime, or service extraction | [modular-monolith-vs-service-extraction.md](modular-monolith-vs-service-extraction.md) | Apply the complete extraction test. |
| Request path/queue, saga, orchestration/choreography, or workflow engine | [sync-async-workflow-ownership.md](sync-async-workflow-ownership.md) | Name process owner, pivot, durable state, and completion model before tools. |
| CQRS, replicas, read service, projection, search, dashboard, export, or aggregator | [read-write-topology-and-projections.md](read-write-topology-and-projections.md) | Preserve write authority and define freshness, bypass, rebuild, and correction. |
| Provider/webhook vocabulary or ambiguous partner result affects lifecycle | [external-provider-anti-corruption.md](external-provider-anti-corruption.md) | Normalize provider evidence behind a local owner. |
| Authority move, extraction, mixed versions, canary, shadow, bridge, or rollback boundary | [rollout-and-migration-patterns.md](rollout-and-migration-patterns.md) | Select target authority first and bound transition machinery. |
| Microservice, distributed-monolith, shared-DB, direct-read, dual-write, retry-storm, fallback, or permanent-shim smell | [architecture-anti-patterns.md](architecture-anti-patterns.md) | Convert the smell into a failure consequence, blocker, risk, or reopen condition. |
