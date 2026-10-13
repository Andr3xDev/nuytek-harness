# Replace Constructor with Factory Method

## Problem

You have a complex constructor that does something more than just setting parameter values in object fields.

## Solution

Create a factory method and use it to replace constructor calls.

## How to Refactor

1. Create a factory method. Place a call to the current constructor in it.

2. Replace all constructor calls with calls to the factory method.

3. Declare the constructor private.

4. Investigate the constructor code and try to isolate the code not directly related to constructing an object of the current class, moving such code to the factory method.

### Helps other refactorings

- Change Value to Reference
- Replace Type Code with Subclasses

### Implements design pattern

- Factory Method
