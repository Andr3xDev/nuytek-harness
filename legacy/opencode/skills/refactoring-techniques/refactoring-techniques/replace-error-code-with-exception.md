# Replace Error Code with Exception

## Problem

A method returns a special value that indicates an error?

## Solution

Throw an exception instead.

## How to Refactor

Try to perform these refactoring steps for only one error code at a time. This will make it easier to keep all the important information in your head and avoid errors.

1. Find all calls to a method that returns error codes and, instead of checking for an error code, wrap it in `try/except` blocks.

2. Inside the method, instead of returning an error code, throw an exception.

3. Change the method signature so that it contains information about the exception being thrown (raise section).
