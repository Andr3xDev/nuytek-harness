# Push Down Field

## Problem

Is a field used only in a few subclasses?

## Solution

Move the field to these subclasses.

## How to Refactor

1. Declare a field in all the necessary subclasses.

2. Remove the field from the superclass.

### Anti-refactoring

- Pull Up Field

### Similar refactorings

- Push Down Method

### Helps other refactorings

- Extract Subclass

### Eliminates smell

- Refused Bequest
