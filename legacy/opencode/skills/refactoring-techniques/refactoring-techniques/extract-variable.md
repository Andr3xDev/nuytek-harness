# Extract Variable

## Problem

You have an expression that’s hard to understand.

## Solution

Place the result of the expression or its parts in separate variables that are self-explanatory.

## How to Refactor

1. Insert a new line before the relevant expression and declare a new variable there. Assign part of the complex expression to this variable.
2. Replace that part of the expression with the new variable.
3. Repeat the process for all complex parts of the expression.

### Anti-refactoring

- Inline Temp

### Similar refactorings

- Extract Method

### Eliminates smell

- Comments
