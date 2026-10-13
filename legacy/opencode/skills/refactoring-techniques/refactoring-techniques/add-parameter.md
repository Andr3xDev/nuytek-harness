# Add Parameter

## Problem

A method doesn’t have enough data to perform certain actions.

## Solution

Create a new parameter to pass the necessary data.

## How to Refactor

1. See whether the method is defined in a superclass or subclass. If the method is present in them, you will need to repeat all the steps in these classes as well.

2. The following step is critical for keeping your program functional during the refactoring process. Create a new method by copying the old one and add the necessary parameter to it. Replace the code for the old method with a call to the new method. You can plug in any value to the new parameter (such as `null` for objects or a zero for numbers).

3. Find all references to the old method and replace them with references to the new method.

4. Delete the old method. Deletion isn’t possible if the old method is part of the public interface. If that’s the case, mark the old method as deprecated.

### Anti-refactoring

- Remove Parameter

### Similar refactorings

- Rename Method

### Helps other refactorings

- Introduce Parameter Object
