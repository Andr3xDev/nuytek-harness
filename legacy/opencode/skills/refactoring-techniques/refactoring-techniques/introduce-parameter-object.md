# Introduce Parameter Object

## Problem

Your methods contain a repeating group of parameters.

## Solution

Replace these parameters with an object.

## How to Refactor

1. Create a new class that will represent your group of parameters. Make the class immutable.

2. In the method that you want to refactor, use **Add Parameter**, which is where your parameter object will be passed. In all method calls, pass the object created from old method parameters to this parameter.

3. Now start deleting old parameters from the method one by one, replacing them in the code with fields of the parameter object. Test the program after each parameter replacement.

4. When done, see whether there’s any point in moving a part of the method (or sometimes even the whole method) to a parameter object class. If so, use **Move Method** or **Extract Method**.

### Similar refactorings

- Preserve Whole Object

### Eliminates smell

- Long Parameter List
- Data Clumps
- Primitive Obsession
- Long Method
