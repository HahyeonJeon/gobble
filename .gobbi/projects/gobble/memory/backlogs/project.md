# project backlog

## Temp driver cleanup on interrupt during compile

**Backlogged at:** 2026-08-20T05:09:58Z

**What:** Remove leftover generated driver files if the user interrupts while a graph verb is compiling.

**Why backlogged:** Wrap-up-readiness iteration 2 accepted this as a non-blocker.

**Context:** The leak is during compile, before the driver runs. Occupancy and signal tests do not cover it.
