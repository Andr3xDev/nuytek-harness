# Consolidate Duplicate Conditional Fragments

## Problem

Identical code can be found in all branches of a conditional.

## Solution

Move the code outside of the conditional.

## How to Refactor

1. If the duplicated code is at the beginning of the conditional branches, move the code to a place before the conditional.

2. If the code is executed at the end of the branches, place it after the conditional.

3. If the duplicate code is randomly situated inside the branches, first try to move the code to the beginning or end of the branch, depending on whether it changes the result of the subsequent code.

4. If appropriate and the duplicate code is longer than one line, try using **Extract Method**.

### Eliminates smell

- Duplicate Code
