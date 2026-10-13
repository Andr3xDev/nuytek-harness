# Replace Temp with Query

## Problem

You place the result of an expression in a local variable for later use in your code.

## Solution

Move the entire expression to a separate method and return the result from it. Query the method instead of using a variable. Incorporate the new method in other methods, if necessary.

## How to Refactor

1. Make sure that a value is assigned to the variable once and only once within the method. If not, use Split Temporary Variable to ensure that the variable will be used only to store the result of your expression.
2. Use Extract Method to place the expression of interest in a new method. Make sure that this method only returns a value and doesn’t change the state of the object. If the method affects the visible state of the object, use Separate Query from Modifier.
3. Replace the variable with a query to your new method.

### Similar refactorings

- Extract Method

### Eliminates smell

- Long Method
- Duplicate Code
