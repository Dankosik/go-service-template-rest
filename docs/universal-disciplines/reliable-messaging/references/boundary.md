# Messaging Boundary

Load when the guarantee, identity, durable commits, acknowledgement, or effect
semantics can change the decision.

Write the guarantee as `<semantic> from <start> after <durable evidence>
through <end>; <effect> is idempotent by <key and scope>`. Name loss/duplicate
tolerance, latency/outage window, fan-out, replay, ordering scope, throughput,
and current evidence. Use a broker only when durable buffering, failure
isolation, independent consumers, fan-out, smoothing, or replay earns its cost.

Give one logical message a producer identity that survives publish retry,
redelivery, redrive, and replay. Keep broker delivery IDs separate. Scope
idempotency to tenant plus consumer/effect and stable message/business identity;
choose retention from the longest delivery, redrive, offline recovery, and
replay horizon.

Commit business state and outbox intent together when possible. Mark published
only after durable broker acceptance; a lost response reuses the same identity.
On consume, atomically commit the business effect and inbox/idempotency record,
or propagate the stable effect key to the external owner. Acknowledge, delete,
or commit offset only after the effect commits. A crash after effect commit and
before acknowledgement must converge on the recorded result.

Falsify both sides of every durable commit, including lost acknowledgement and
concurrent duplicate delivery. Reopen the domain owner when message meaning,
effect identity, or allowed state transition is unsettled.
