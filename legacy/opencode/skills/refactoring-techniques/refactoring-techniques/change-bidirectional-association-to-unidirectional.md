# Change Bidirectional Association to Unidirectional

## Problem

You have a bidirectional association between classes, but one of the classes doesn’t use the other’s features.

## Solution

Remove the unused association.

## How to Refactor

1. Make sure that one of the following is true for your classes:

   - No association is used.

   - There’s another way to get the associated object, such through a database query.

   - The associated object can be passed as an argument to the methods that use it.

2. Depending on your situation, use of a field that contains an association with another object should be replaced by a parameter or method call for getting the object in a different way.

3. Delete the code that assigns the associated object to the field.

4. Delete the now-unused field.

### Anti-refactoring

- Change Unidirectional Association to Bidirectional

### Eliminates smell

- Inappropriate Intimacy
