# Remove Middle Man

## Problem

A class has too many methods that simply delegate to other objects.

## Solution

Delete these methods and force the client to call the end methods directly.

## How to Refactor

1. Create a getter for accessing the delegate-class object from the server-class object.

2. Replace calls to delegating methods in the server-class with direct calls for methods in the delegate-class.

### Anti-refactoring

- Hide Delegate

### Eliminates smell

- Middle Man
