# Replace Type Code with State/Strategy

What’s type code? Type code occurs when, instead of a separate data type, you have a set of numbers or strings that form a list of allowable values for some entity. Often these specific numbers and strings are given understandable names via constants, which is the reason for why such type code is encountered so much.

## Problem

You have a coded type that affects behavior but you can’t use subclasses to get rid of it.

## Solution

Replace type code with a state object. If it’s necessary to replace a field value with type code, another state object is “plugged in”.

## How to Refactor

1. Use **Self Encapsulate Field** to create a getter for the field that contains type code.

2. Create a new class and give it an understandable name that fits the purpose of the type code. This class will be playing the role of state (or strategy). In it, create an abstract coded field getter.

3. Create subclasses of the state class for each value of the coded type. In each subclass, redefine the getter of the coded field so that it returns the corresponding value of the coded type.

4. In the abstract state class, create a static factory method that accepts the value of the coded type as a parameter. Depending on this parameter, the factory method will create objects of various states. For this, in its code create a large conditional; it’ll be the only one when refactoring is complete.

5. In the original class, change the type of the coded field to the state class. In the field’s setter, call the factory state method for getting new state objects.

6. Now you can start to move the fields and methods from the superclass to the corresponding state subclasses (using **Push Down Field** and **Push Down Method**).

7. When everything moveable has been moved, use **Replace Conditional with Polymorphism** in order to get rid of conditionals that use type code once and for all.

### Similar refactorings

- Replace Type Code with Class
- Replace Type Code with Subclasses

### Implements design pattern

- State
- Strategy

### Eliminates smell

- Primitive Obsession
