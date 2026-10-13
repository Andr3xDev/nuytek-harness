# Inline Temp

## Problem

You have a temporary variable that’s assigned the result of a simple expression and nothing more.

## Solution

Replace the references to the variable with the expression itself.

## How to Refactor

1. Find all places that use the variable. Instead of the variable, use the expression that had been assigned to it.
2. Delete the declaration of the variable and its assignment line.

### Helps other refactorings

- Replace Temp with Query
- Extract Method
