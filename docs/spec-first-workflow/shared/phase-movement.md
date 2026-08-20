# Phase Movement

Load when moving or reopening a macro phase.

Move only when the current owner returns [Phase Result
V1](../interfaces/phase-result-v1.md) as `ready`, every triggered decision has a
disposition, and the next owner can act without inventing meaning, mechanism,
ownership, proof strategy, or authority.

Reopen only the smallest owner invalidated by current evidence and preserve
unaffected decisions and proof. Supporting research, review, repair, and that
narrow reopen stay inside the active macro phase. Stop at an explicit named
phase boundary, unavailable required external input, new authority boundary,
or required durable handoff.
