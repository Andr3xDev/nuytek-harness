# Replace Array with Object

This refactoring technique is a special case of **Replace Data Value with Object**.

## Problem

You have an array that contains various types of data.

## Solution

Replace the array with an object that will have separate fields for each element.

## How to Refactor

1. Create the new class that will contain the data from the array. Place the array itself in the class as a public field.

2. Create a field for storing the object of this class in the original class. Don’t forget to also create the object itself in the place where you initiated the data array.

3. In the new class, create access methods one by one for each of the array elements. Give them self-explanatory names that indicate what they do. At the same time, replace each use of an array element in the main code with the corresponding access method.

4. When access methods have been created for all elements, make the array private.

5. For each element of the array, create a private field in the class and then change the access methods so that they use this field instead of the array.

6. When all data has been moved, delete the array.

### Similar refactorings

- Replace Data Value with Object

### Eliminates smell

- Primitive Obsession
