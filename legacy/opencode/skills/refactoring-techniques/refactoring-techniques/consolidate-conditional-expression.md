# Consolidate Conditional Expression

## Problem

You have multiple conditionals that lead to the same result or action.

## Solution

Consolidate all these conditionals in a single expression.

## How to Refactor

Before refactoring, make sure that the conditionals don’t have any “side effects” or otherwise modify something, instead of simply returning values. Side effects may be hiding in the code executed inside the operator itself, such as when something is added to a variable based on the results of a conditional.

1. Consolidate the conditionals in a single expression by using `and` and `or`. As a general rule when consolidating:

   - Nested conditionals are joined using `and`.

   - Consecutive conditionals are joined with `or`.

2. Perform **Extract Method** on the operator conditions and give the method a name that reflects the expression’s purpose.

### Eliminates smell

- Duplicate Code
