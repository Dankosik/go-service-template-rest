# Incident Localization

Load when the production symptom or owning hop is unresolved.

Contract the symptom as metric and unit, current value versus a comparable
baseline, affected cohort, onset, and shape; record the unaffected complement.
Bound user impact and trend. Before any restart, failover, or other
evidence-destroying action, preserve perishable dumps, queues, connections, and
recent logs that can change localization.

Build a time-aligned change set around onset: deploys, config/flags,
dependencies, certificates/quotas/keys, scheduled jobs, traffic mix, and data
growth. Correlation nominates a hypothesis only when a mechanism can explain
the observed cohort and shape.

Walk the path from edge inward. For latency compare self-time and downstream
time; for errors find the originating fingerprint before wrappers; for
saturation compare utilization, queue, and failure at each resource; for skew
compare instances, shards, tenants, keys, and versions. Order changes by onset
to distinguish the first mover from louder downstream victims. Missing
telemetry calls for the smallest safe discriminating probe, not blind tuning.

Reopen the matching domain owner with the symptom contract, timeline, and
localization evidence once the first owning boundary is known.
