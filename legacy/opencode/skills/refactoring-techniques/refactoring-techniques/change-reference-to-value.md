# Change Reference to Value

## Problem

You have a reference object that’s too small and infrequently changed to justify managing its life cycle.

## Solution

Turn it into a value object.

## How to Refactor

1. Make the object unchangeable. The object shouldn’t have any setters or other methods that change its state and data (**Remove Setting Method** may help here). The only place where data should be assigned to the fields of a value object is a constructor.

2. Create a comparison method to be able to compare two values.

3. Check whether you can delete the factory method and make the object constructor public.

### Anti-refactoring

- Change Value to Reference
