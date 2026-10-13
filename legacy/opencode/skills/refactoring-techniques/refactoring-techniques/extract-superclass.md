# Extract Superclass

## Problem

You have two classes with common fields and methods.

## Solution

Create a shared superclass for them and move all the identical fields and methods to it.

## When Not to Use

You can not apply this technique to classes that already have a superclass.

## How to Refactor

1. Create an abstract superclass.

2. Use **Pull Up Field**, **Pull Up Method**, and **Pull Up Constructor Body** to move the common functionality to a superclass. Start with the fields, since in addition to the common fields you will need to move the fields that are used in the common methods.

3. Look for places in the client code where use of subclasses can be replaced with your new class (such as in type declarations).

### Similar refactorings

- Extract Interface

### Eliminates smell

- Duplicate Code
