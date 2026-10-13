# Replace Type Code with Subclasses

What’s type code? Type code occurs when, instead of a separate data type, you have a set of numbers or strings that form a list of allowable values for some entity. Often these specific numbers and strings are given understandable names via constants, which is the reason for why such type code is encountered so much.

## Problem

You have a coded type that directly affects program behavior (values of this field trigger various code in conditionals).

## Solution

Create subclasses for each value of the coded type. Then extract the relevant behaviors from the original class to these subclasses. Replace the control flow code with polymorphism.

## How to Refactor

1. Use **Self Encapsulate Field** to create a getter for the field that contains type code.

2. Make the superclass constructor private. Create a static factory method with the same parameters as the superclass constructor. It must contain the parameter that will take the starting values of the coded type. Depending on this parameter, the factory method will create objects of various subclasses. To do so, in its code you must create a large conditional but, at least, it’ll be the only one when it’s truly necessary; otherwise, subclasses and polymorphism will do.

3. Create a unique subclass for each value of the coded type. In it, redefine the getter of the coded type so that it returns the corresponding value of the coded type.

4. Delete the field with type code from the superclass. Make its getter abstract.

5. Now that you have subclasses, you can start to move the fields and methods from the superclass to corresponding subclasses (with the help of **Push Down Field** and **Push Down Method**).

6. When everything possible has been moved, use **Replace Conditional with Polymorphism** in order to get rid of conditions that use the type code once and for all.

### Anti-refactoring

- Replace Subclass with Fields

### Similar refactorings

- Replace Type Code with Class
- Replace Type Code with State/Strategy

### Eliminates smell

- Primitive Obsession
