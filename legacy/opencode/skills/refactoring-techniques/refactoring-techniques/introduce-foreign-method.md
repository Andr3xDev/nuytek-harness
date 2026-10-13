# Introduce Foreign Method

## Problem

A utility class doesn’t contain the method that you need and you can’t add the method to the class.

## Solution

Add the method to a client class and pass an object of the utility class to it as an argument.

## How to Refactor

1. Create a new method in the client class.
2. In this method, create a parameter to which the object of the utility class will be passed. If this object can be obtained from the client class, you don’t have to create such a parameter.
3. Extract the relevant code fragments to this method and replace them with method calls.
4. Be sure to leave the Foreign method tag in the comments for the method along with the advice to place this method in a utility class if such becomes possible later. This will make it easier to understand why this method is located in this particular class for those who’ll be maintaining the software in the future.

### Similar refactorings

- Introduce Local Extension (Move all extension methods to a separate class, which is wrapper or a subclass of some service class).

### Eliminates smell

- Incomplete Library Class
