# Pull Up Field

## Problem

Two classes have the same field.

## Solution

Remove the field from subclasses and move it to the superclass.

## How to Refactor

1. Make sure that the fields are used for the same needs in subclasses.

2. If the fields have different names, give them the same name and replace all references to the fields in existing code.

3. Create a field with the same name in the superclass. Note that if the fields were private, the superclass field should be protected.

4. Remove the fields from the subclasses.

5. You may want to consider using **Self Encapsulate Field** for the new field, in order to hide it behind access methods.

### Anti-refactoring

- Push Down Field

### Similar refactorings

- Pull Up Method

### Eliminates smell

- Duplicate Code
