# Replace Conditional with Polymorphism

## Problem

You have a conditional that performs various actions depending on object type or properties.

## Solution

Create subclasses matching the branches of the conditional. In them, create a shared method and move code from the corresponding branch of the conditional to it. Then replace the conditional with the relevant method call. The result is that the proper implementation will be attained via polymorphism depending on the object class.

## How to Refactor

### Preparing to Refactor

For this refactoring technique, you should have a ready hierarchy of classes that will contain alternative behaviors. If you don’t have a hierarchy like this, create one. Other techniques will help to make this happen:

- **Replace Type Code with Subclasses**. Subclasses will be created for all values of a particular object property. This approach is simple but less flexible since you can’t create subclasses for the other properties of the object.

- **Replace Type Code with State/Strategy**. A class will be dedicated for a particular object property and subclasses will be created from it for each value of the property. The current class will contain references to the objects of this type and delegate execution to them.

The following steps assume that you have already created the hierarchy.

### Refactoring Steps

1. If the conditional is in a method that performs other actions as well, perform **Extract Method**.

2. For each hierarchy subclass, redefine the method that contains the conditional and copy the code of the corresponding conditional branch to that location.

3. Delete this branch from the conditional.

4. Repeat replacement until the conditional is empty. Then delete the conditional and declare the method abstract.

### Eliminates smell

- Switch Statements
