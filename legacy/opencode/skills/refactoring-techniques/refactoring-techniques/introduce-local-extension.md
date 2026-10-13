# Introduce Local Extension

## Problem

A utility class doesn’t contain some methods that you need. But you can’t add these methods to the class.

## Solution

Create a new class containing the methods and make it either the child or wrapper of the utility class.

## How to Refactor

1. Create a new extension class:

   - Option A: Make it a child of the utility class.

   - Option B: If you have decided to make a wrapper, create a field in it for storing the utility class object to which delegation will be made. When using this option, you will need to also create methods that repeat the public methods of the utility class and contain simple delegation to the methods of the utility object.

2. Create a constructor that uses the parameters of the constructor of the utility class.

3. Also create an alternative “converting” constructor that takes only the object of the original class in its parameters. This will help to substitute the extension for the objects of the original class.

4. Create new extended methods in the class. Move foreign methods from other classes to this class or else delete the foreign methods if their functionality is already present in the extension.

5. Replace use of the utility class with the new extension class in places where its functionality is needed.

### Similar refactorings

- Introduce Foreign Method: If you only want one special method, which doesn't exist in service class, and you can't extend it, move it to the client class and pass the object of a service class as a parameter.

### Eliminates smell

Incomplete Library Class
