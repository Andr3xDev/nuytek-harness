# Replace Type Code with Class

What’s type code? Type code occurs when, instead of a separate data type, you have a set of numbers or strings that form a list of allowable values for some entity. Often these specific numbers and strings are given understandable names via constants, which is the reason for why such type code is encountered so much.

## Problem

A class has a field that contains type code. The values of this type aren’t used in operator conditions and don’t affect the behavior of the program.

## Solution

Create a new class and use its objects instead of the type code values.

## How to Refactor

1. Create a new class and give it a new name that corresponds to the purpose of the coded type. Here we’ll call it type class.

2. Copy the field containing type code to the type class and make it private. Then create a getter for the field. A value will be set for this field only from the constructor.

3. For each value of the coded type, create a static method in type class. It’ll be creating a new type class object corresponding to this value of the coded type.

4. In the original class, replace the type of the coded field with type class. Create a new object of this type in the constructor as well as in the field setter. Change the field getter so that it calls the type class getter.

5. Replace any mentions of values of the coded type with calls of the relevant type class static methods.

6. Remove the coded type constants from the original class.

### Similar refactorings

- Replace Type Code with Subclasses
- Replace Type Code with State/Strategy

### Eliminates smell

- Primitive Obsession
