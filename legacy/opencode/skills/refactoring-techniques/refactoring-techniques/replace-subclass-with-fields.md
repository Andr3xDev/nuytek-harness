# Replace Subclass with Fields

## Problem

You have subclasses differing only in their (constant-returning) methods.

## Solution

Replace the methods with fields in the parent class and delete the subclasses.

## How to Refactor

1. Apply **Replace Constructor with Factory Method** to the subclasses.

2. Replace subclass constructor calls with superclass factory method calls.

3. In the superclass, declare fields for storing the values of each of the subclass methods that return constant values.

4. Create a protected superclass constructor for initializing the new fields.

5. Create or modify the existing subclass constructors so that they call the new constructor of the parent class and pass the relevant values to it.

6. Implement each constant method in the parent class so that it returns the value of the corresponding field. Then remove the method from the subclass.

7. If the subclass constructor has additional functionality, use **Inline Method** to incorporate the constructor into the superclass factory method.

8. Delete the subclass.

### Anti-refactoring

- Replace Type Code with Subclasses
