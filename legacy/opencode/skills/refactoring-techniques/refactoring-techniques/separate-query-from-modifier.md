# Separate Query from Modifier

## Problem

Do you have a method that returns a value but also changes something inside an object?

## Solution

Split the method into two separate methods. As you would expect, one of them should return the value and the other one modifies the object.

## How to Refactor

1. Create a new query method to return what the original method did.

2. Change the original method so that it returns only the result of calling the new query method.

3. Replace all references to the original method with a call to the query method. Immediately before this line, place a call to the modifier method. This will save you from side effects in case if the original method was used in a condition of a conditional operator or loop.

4. Get rid of the value-returning code in the original method, which now has become a proper modifier method.

### Helps other refactorings

- Replace Temp with Query
