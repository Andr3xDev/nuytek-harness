---
name: event-schema
description: "Trigger: domain event, event schema, event-driven, emits event. Validate that domain events are defined before implementation."
license: MIT
metadata:
  author: custom
  version: "1.0"
---

## Activation Contract

Use this skill when a feature emits, consumes, or modifies domain events.
The orchestrator injects this into sdd-spec and sdd-design when the proposal
mentions events. Also triggered by the SDD gate when events are involved.

## Hard Rules

- No implementation of event-emitting code without a defined schema.
- Event schemas must specify: name, payload shape, producer, consumer(s), and version.
- New events must be registered in the project's domain events catalog.
- Changing an existing event's payload is a breaking change — flag it as CRITICAL.

## Decision Gates

| Situation | Action |
|---|---|
| New event needed | Define schema, register in catalog, proceed |
| Existing event changed | Flag as CRITICAL breaking change, ask for approval |
| Event consumed but not defined | Block — find or define the producer's schema first |
| Unsure if feature needs events | Check the proposal and design — if cross-service, likely yes |

## Execution Steps

1. Read the proposal and spec for mentions of events, messages, signals, or cross-service communication.
2. For each event identified:
   - Define the schema:
     ```
     event: MessageSent
     version: 1
     producer: bridge-api
     consumers: [notification-service, analytics-service]
     payload:
       message_id: string (UUID)
       conversation_id: string (UUID)
       sender_id: string (UUID)
       timestamp: string (ISO 8601)
     ```
   - Verify the event name follows project naming conventions
   - Check if the event already exists in the catalog — if yes, validate compatibility
3. Register new events in the project's domain events catalog (context repo or CLAUDE.md).
4. Include event schema in the spec artifacts so test-writer can write contract tests.

## Output Contract

Return for each event:
- Event name and version
- Schema definition
- Producer and consumers
- Whether it's new or modified
- If modified: breaking change assessment
