# Code Smells

## Bloaters

Bloaters are code, methods and classes that have increased to such gargantuan proportions that they are hard to work with. Usually these smells do not crop up right away, rather they accumulate over time as the program evolves (and especially when nobody makes an effort to eradicate them).

- [Long Method](./code-smells/long-method.md)
- [Large Class](./code-smells/large-class.md)
- [Primitive Obsession](./code-smells/primitive-obsession.md)
- [Long Parameter List](./code-smells/long-parameter-list.md)
- [Data Clumps](./code-smells/data-clumps.md)

## Object-Orientation Abusers

All these smells are incomplete or incorrect application of object-oriented programming principles.

- [Switch Statements](./code-smells/switch-statements.md)
- [Temporary Field](./code-smells/temporary-field.md)
- [Refused Bequest](./code-smells/refused-bequest.md)
- [Alternative Classes with Different Interfaces](./code-smells/alternative-classes-with-different-interfaces.md)

## Change Preventers

These smells mean that if you need to change something in one place in your code, you have to make many changes in other places too. Program development becomes much more complicated and expensive as a result.

- [Divergent Change](./code-smells/divergent-change.md)
- [Parallel Inheritance Hierarchies](./code-smells/parallel-inheritance-hierarchies.md)
- [Shotgun Surgery](./code-smells/shotgun-surgery.md)

## Dispensables

A dispensable is something pointless and unneeded whose absence would make the code cleaner, more efficient and easier to understand.

- [Comments](./code-smells/comments.md)
- [Duplicate Code](./code-smells/duplicate-code.md)
- [Data Class](./code-smells/data-class.md)
- [Dead Code](./code-smells/dead-code.md)
- [Lazy Class](./code-smells/lazy-class.md)
- [Speculative Generality](./code-smells/speculative-generality.md)

## Couplers

All the smells in this group contribute to excessive coupling between classes or show what happens if coupling is replaced by excessive delegation.

- [Feature Envy](./code-smells/feature-envy.md)
- [Inappropriate Intimacy](./code-smells/inappropriate-intimacy.md)
- [Incomplete Library Class](./code-smells/incomplete-library-class.md)
- [Message Chains](./code-smells/message-chains.md)
- [Middle Man](./code-smells/middle-man.md)
