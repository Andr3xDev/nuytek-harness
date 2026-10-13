# Replace Delegation with Inheritance

## Problem

A class contains many simple methods that delegate to all methods of another class.

## Solution

Make the class a delegate inheritor, which makes the delegating methods unnecessary.

## When Not to Use

Don’t use this technique if the class contains delegation to only a portion of the public methods of the delegate class. By doing so, you would violate the Liskov substitution principle.

This technique can be used only if the class still doesn’t have parents.

## How to Refactor

1. Make the class a subclass of the delegate class.

2. Place the current object in a field containing a reference to the delegate object.

3. Delete the methods with simple delegation one by one. If their names were different, use **Rename Method** to give all the methods a single name.

4. Replace all references to the delegate field with references to the current object.

5. Remove the delegate field.

### Anti-refactoring

- Replace Inheritance with Delegation

### Similar refactorings

- Remove Middle Man

### Eliminates smell

- Inappropriate Intimacy
