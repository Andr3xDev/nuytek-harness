# Push Down Method

## Problem

Is behavior implemented in a superclass used by only one (or a few) subclasses?

## Solution

Move this behavior to the subclasses.

## How to Refactor

1. Declare the method in a subclass and copy its code from the superclass.

2. Remove the method from the superclass.

3. Find all places where the method is used and verify that it’s called from the necessary subclass.

### Anti-refactoring

- Pull Up Method

### Similar refactorings

- Push Down Field

### Helps other refactorings

- Extract Subclass

### Eliminates smell

- Refused Bequest
