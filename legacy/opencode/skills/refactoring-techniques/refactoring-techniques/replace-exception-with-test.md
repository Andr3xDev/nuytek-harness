# Replace Exception with Test

## Problem

You throw an exception in a place where a simple test would do the job?

## Solution

Replace the exception with a condition test.

## How to Refactor

1. Create a conditional for an edge case and move it before the try/catch block.

2. Move code from the `except` section inside this conditional.

3. In the `except` section, place the code for throwing a usual unnamed exception and run all the tests.

4. If no exceptions were thrown during the tests, get rid of the `try/except` operator.

### Similar refactorings

- Replace Error Code with Exception
