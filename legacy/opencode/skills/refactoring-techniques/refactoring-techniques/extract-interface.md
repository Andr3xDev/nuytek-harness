# Extract Interface

## Problem

Multiple clients are using the same part of a class interface. Another case: part of the interface in two classes is the same.

## Solution

Move this identical portion to its own interface.

## How to Refactor

1. Create an empty interface.

2. Declare common operations in the interface.

3. Declare the necessary classes as implementing the interface.

4. Change type declarations in the client code to use the new interface.

### Similar refactorings

- Extract Superclass
