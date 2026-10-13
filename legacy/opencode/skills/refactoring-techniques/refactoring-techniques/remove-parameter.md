# Remove Parameter

## Problem

A parameter isn’t used in the body of a method.

## Solution

Remove the unused parameter.

## When Not to Use

If the method is implemented in different ways in subclasses or in a superclass, and your parameter is used in those implementations, leave the parameter as-is.

## How to Refactor

1. See whether the method is defined in a superclass or subclass. If so, is the parameter used there? If the parameter is used in one of these implementations, hold off on this refactoring technique.

2. The next step is important for keeping the program functional during the refactoring process. Create a new method by copying the old one and delete the relevant parameter from it. Replace the code of the old method with a call to the new one.

3. Find all references to the old method and replace them with references to the new method.

4. Delete the old method. Don’t perform this step if the old method is part of a public interface. In this case, mark the old method as deprecated.

### Anti-refactoring

- Add Parameter

### Similar refactorings

- Rename Method

### Helps other refactorings

- Replace Parameter with Method Call

### Eliminates smell

- Speculative Generality
