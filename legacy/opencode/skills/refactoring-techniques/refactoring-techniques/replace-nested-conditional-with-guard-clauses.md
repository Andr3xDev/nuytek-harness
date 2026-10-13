# Replace Nested Conditional with Guard Clauses

## Problem

You have a group of nested conditionals and it’s hard to determine the normal flow of code execution.

## Solution

Isolate all special checks and edge cases into separate clauses and place them before the main checks. Ideally, you should have a “flat” list of conditionals, one after the other.

## How to Refactor

1. Try to rid the code of side effects—**Separate Query from Modifier** may be helpful for the purpose. This solution will be necessary for the reshuffling described below.

2. Isolate all guard clauses that lead to calling an exception or immediate return of a value from the method. Place these conditions at the beginning of the method.

3. After rearrangement is complete and all tests are successfully completed, see whether you can use **Consolidate Conditional Expression** for guard clauses that lead to the same exceptions or returned values.
