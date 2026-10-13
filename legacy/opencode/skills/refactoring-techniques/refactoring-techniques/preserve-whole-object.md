# Preserve Whole Object

## Problem

You get several values from an object and then pass them as parameters to a method.

## Solution

Instead, try passing the whole object.

## How to Refactor

1. Create a parameter in the method for the object from which you can get the necessary values.

2. Now start removing the old parameters from the method one by one, replacing them with calls to the relevant methods of the parameter object. Test the program after each replacement of a parameter.

3. Delete the getter code from the parameter object that had preceded the method call.

### Similar refactorings

- Introduce Parameter Object
- Replace Parameter with Method Call

### Eliminates smell

- Primitive Obsession
- Long Parameter List
- Long Method
- Data Clumps
