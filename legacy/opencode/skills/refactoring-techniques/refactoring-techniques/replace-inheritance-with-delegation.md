# Replace Inheritance with Delegation

## Problem

You have a subclass that uses only a portion of the methods of its superclass (or it’s not possible to inherit superclass data).

## Solution

Create a field and put a superclass object in it, delegate methods to the superclass object, and get rid of inheritance.

## How to Refactor

1. Create a field in the subclass for holding the superclass. During the initial stage, place the current object in it.

2. Change the subclass methods so that they use the superclass object instead of `self`.

3. For methods inherited from the superclass that are called in the client code, create simple delegating methods in the subclass.

4. Remove the inheritance declaration from the subclass.

5. Change the initialization code of the field in which the former superclass is stored by creating a new object.

### Anti-refactoring

- Replace Delegation with Inheritance

### Implements design pattern

- Strategy
