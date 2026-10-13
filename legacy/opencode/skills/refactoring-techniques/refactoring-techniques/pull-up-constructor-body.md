# Pull Up Constructor Body

## Problem

Your subclasses have constructors with code that’s mostly identical.

## Solution

Create a superclass constructor and move the code that’s the same in the subclasses to it. Call the superclass constructor in the subclass constructors.

## How to Refactor

1. Create a constructor in a superclass.

2. Extract the common code from the beginning of the constructor of each subclass to the superclass constructor. Before doing so, try to move as much common code as possible to the beginning of the constructor.

3. Place the call for the superclass constructor in the first line in the subclass constructors.

## Similar refactorings

- Pull Up Method

## Eliminates smell

- Duplicate Code
