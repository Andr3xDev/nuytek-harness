# Replace Method with Method Object

## Problem

You have a long method in which the local variables are so intertwined that you can’t apply Extract Method.

## Solution

Transform the method into a separate class so that the local variables become fields of the class. Then you can split the method into several methods within the same class.

## How to Refactor

1. Create a new class. Name it based on the purpose of the method that you’re refactoring.
2. In the new class, create a private field for storing a reference to an instance of the class in which the method was previously located. It could be used to get some required data from the original class if needed.
3. Create a separate private field for each local variable of the method.
4. Create a constructor that accepts as parameters the values of all local variables of the method and also initializes the corresponding private fields.
5. Declare the main method and copy the code of the original method to it, replacing the local variables with private fields.
6. Replace the body of the original method in the original class by creating a method object and calling its main method.

### Similar refactorings

- Replace Data Value with Object (Does the same with fields).

### Eliminates smell

- Long Method
