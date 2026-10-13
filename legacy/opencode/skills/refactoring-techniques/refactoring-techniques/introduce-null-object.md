# Introduce Null Object

## Problem

Since some methods return null instead of real objects, you have many checks for null in your code.

## Solution

Instead of null, return a null object that exhibits the default behavior.

## How to Refactor

1. From the class in question, create a subclass that will perform the role of null object.

2. In both classes, create the method `is_null()`, which will return `true` for a null object and `false` for a real class.

3. Find all places where the code may return `null` instead of a real object. Change the code so that it returns a null object.

4. Find all places where the variables of the real class are compared with `null`. Replace these checks with a call for `is_null()`.

5. - If methods of the original class are run in these conditionals when a value doesn’t equal `null`, redefine these methods in the null class and insert the code from the `else` part of the condition there. Then you can delete the entire conditional and differing behavior will be implemented via polymorphism.
   - If things aren’t so simple and the methods can’t be redefined, see if you can simply extract the operators that were supposed to be performed in the case of a `null` value to new methods of the null object. Call these methods instead of the old code in `else` as the operations by default.

### Similar refactorings

- Replace Conditional with Polymorphism

### Implements design pattern

- Null-object

### Eliminates smell

- Switch Statements
- Temporary Field
