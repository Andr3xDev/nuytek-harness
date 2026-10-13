---
name: software-design-patterns
description: Reference skill for software design patterns. Trigger when the user has doubts about which pattern to use, wants to understand a specific pattern, or needs help identifying the right pattern for a problem. This is a consultation tool — identify the relevant pattern, read its file, present findings, and wait for the user's direction.
---
# Software Design Patterns

Typical solutions to common problems in software design. Each pattern is a
blueprint you can customize to solve a particular design problem in your code.

---

## Scope

**Covers:** creational and behavioral design patterns at code level.

**Does NOT cover:** architecture patterns (see `senior-architect`), infrastructure, or framework-specific patterns.

---

## Preconditions

- The user has a design problem, a doubt about which pattern applies, or wants to understand a specific pattern.
- If the question is about system-level architecture (microservices, event-driven, etc.), redirect to `senior-architect`.

---

## Navigation Rule

1. Listen to the user's problem or question.
2. Identify which category and pattern(s) are relevant.
3. Read **only** the specific pattern file — do not load everything.
4. Present: what the pattern does, how it relates to the user's case, and how it connects to other patterns.
5. **Wait for the user's confirmation before writing any code or applying changes.**

---

## Pattern Catalog

### Creational Patterns

Object creation mechanisms that increase flexibility and reuse.

| Pattern | Problem it solves | File |
|---------|-------------------|------|
| Factory Method | Need to create objects without specifying exact class; let subclasses decide | `creational/factory-method.md` |
| Abstract Factory | Need to create families of related objects without coupling to concrete classes | `creational/abstract-factory.md` |
| Builder | Need to construct complex objects step by step with different representations | `creational/builder.md` |
| Prototype | Need to copy existing objects without depending on their classes | `creational/prototype.md` |
| Singleton | Need exactly one instance of a class with a global access point | `creational/singleton.md` |

Index with full descriptions: [creational-patterns.md](creational-patterns.md)

### Behavioral Patterns

Algorithms and assignment of responsibilities between objects.

| Pattern | Problem it solves | File |
|---------|-------------------|------|
| Chain of Responsibility | Need to pass a request along a chain until some handler processes it | `behavioral/chain-of-responsibility.md` |
| Command | Need to turn requests into objects (queue, undo, log operations) | `behavioral/command.md` |
| Iterator | Need to traverse a collection without exposing its internal structure | `behavioral/iterator.md` |
| Mediator | Need to reduce chaotic dependencies by centralizing communication | `behavioral/mediator.md` |
| Memento | Need to save/restore object state without breaking encapsulation | `behavioral/memento.md` |
| Observer | Need to notify multiple objects when something changes | `behavioral/observer.md` |
| State | Need an object to change behavior when its internal state changes | `behavioral/state.md` |
| Strategy | Need to swap algorithms at runtime within the same context | `behavioral/strategy.md` |
| Template Method | Need a skeleton algorithm where subclasses override specific steps | `behavioral/template-method.md` |
| Visitor | Need to add operations to objects without modifying their classes | `behavioral/visitor.md` |

Index with full descriptions: [behavioral-patterns.md](behavioral-patterns.md)

---

## Quick Pattern Finder

When the user describes a problem, use this to narrow down:

| If the user needs to... | Consider |
|--------------------------|----------|
| Create objects flexibly | Factory Method, Abstract Factory, Builder |
| Copy objects | Prototype |
| Ensure single instance | Singleton |
| Decouple sender from receiver | Chain of Responsibility, Mediator |
| Queue/undo operations | Command, Memento |
| React to changes | Observer |
| Swap behavior at runtime | Strategy, State |
| Vary steps of an algorithm | Template Method |
| Add operations without modifying classes | Visitor |
| Traverse collections | Iterator |

---

## Control Points

- **Before reading a pattern file:** confirm with the user which problem they're trying to solve.
- **After reading:** summarize the pattern and how it fits their case. Do not dump the entire file.
- **If multiple patterns could apply:** present the options with the "Relations with Other Patterns" section to clarify differences. Let the user pick.
- **Before writing code:** get explicit approval. Show the structure first.

---

## Constraints

- Do not recommend patterns from general knowledge alone — ground recommendations in the pattern files.
- Do not load all pattern files at once. Read only what's relevant.
- Keep responses concise. Present the pattern, its fit, and wait.
- If the user's problem doesn't match any pattern here, say so — do not force a pattern where none fits.
