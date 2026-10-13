# Inline Method

## Problem

When a method body is more obvious than the method itself, use this technique.

## Solution

Replace calls to the method with the method’s content and delete the method itself.

## How to Refactor

1. Make sure that the method isn’t redefined in subclasses. If the method is redefined, refrain from this technique.
2. Find all calls to the method. Replace these calls with the content of the method.
3. Delete the method.

### Anti-refactoring

- Extract Method

### Eliminates smell

- Speculative Generality
