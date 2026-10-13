# Introduce Assertion

## Problem

For a portion of code to work correctly, certain conditions or values must be true.

## Solution

Replace these assumptions with specific assertion checks.

## How to Refactor

When you see that a condition is assumed, add an assertion for this condition in order to make sure.

Adding the assertion shouldn’t change the program’s behavior.

Don’t overdo it with use of assertions for everything in your code. Check for only the conditions that are necessary for correct functioning of the code. If your code is working normally even when a particular assertion is false, you can safely remove the assertion.

### Eliminates smell

- Comments
