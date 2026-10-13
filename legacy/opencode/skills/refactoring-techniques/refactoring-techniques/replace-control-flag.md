# Remove Control Flag

## Problem

You have a boolean variable that acts as a control flag for multiple boolean expressions.

## Solution

Instead of the variable, use `break`, `continue` and `return`.

## How to Refactor

1. Find the value assignment to the control flag that causes the exit from the loop or current iteration.

2. Replace it with `break`, if this is an exit from a loop; `continue`, if this is an exit from an iteration, or `return`, if you need to return this value from the function.

3. Remove the remaining code and checks associated with the control flag.
