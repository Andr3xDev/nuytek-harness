# Self Encapsulate Field

Self-encapsulation is distinct from ordinary **Encapsulate Field**: the refactoring technique given here is performed on a private field.

## Problem

You use direct access to private fields inside a class.

## Solution

Create a getter and setter for the field, and use only them for accessing the field.

## How to Refactor

1. Create a getter (and optional setter) for the field. They should be either `protected` or `public`.

2. Find all direct invocations of the field and replace them with getter and setter calls.

### Similar refactorings

- Encapsulate Field (Hide public fields, provide getters and setters).

### Helps other refactorings

- Duplicate Observed Data
- Replace Type Code with Subclasses
- Replace Type Code with State/Strategy
