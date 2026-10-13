# Move Field

## Problem

A field is used more in another class than in its own class.

## Solution

Create a field in a new class and redirect all users of the old field to it.

## How to Refactor

1. If the field is public, refactoring will be much easier if you make the field private and provide public access methods (for this, you can use Encapsulate Field).
2. Create the same field with access methods in the recipient class.
3. Decide how you will refer to the recipient class. You may already have a field or method that returns the appropriate object; if not, you will need to write a new method or field to store the object of the recipient class.
4. Replace all references to the old field with appropriate calls to methods in the recipient class. If the field isn’t private, take care of this in the superclass and subclasses.
5. Delete the field in the original class.

### Similar refactorings

- Move Method

### Helps other refactorings

- Extract Class
- Inline Class

### Eliminates smell

- Shotgun Surgery
- Parallel Inheritance Hierarchies
- Inappropriate Intimacy
