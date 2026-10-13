# Split Temporary Variable

## Problem

You have a local variable that’s used to store various intermediate values inside a method (except for cycle variables).

## Solution

Use different variables for different values. Each variable should be responsible for only one particular thing.

## How to Refactor

1. Find the first place in the code where the variable is given a value. Here you should rename the variable with a name that corresponds to the value being assigned.
2. Use the new name instead of the old one in places where this value of the variable is used.
3. Repeat as needed for places where the variable is assigned a different value.

### Anti-refactoring

- Inline Temp

## Similar refactorings

- Extract Variable
- Remove Assignments to Parameters

## Helps other refactorings

- Extract Method
