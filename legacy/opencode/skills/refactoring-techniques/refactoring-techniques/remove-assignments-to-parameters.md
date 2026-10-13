# Remove Assignments to Parameters

## Problem

Some value is assigned to a parameter inside method’s body.

## Solution

Use a local variable instead of a parameter.

## How to Refactor

1. Create a local variable and assign the initial value of your parameter.
2. In all method code that follows this line, replace the parameter with your new local variable.

### Similar refactorings

- Split Temporary Variable

### Helps other refactorings

- Extract Method
