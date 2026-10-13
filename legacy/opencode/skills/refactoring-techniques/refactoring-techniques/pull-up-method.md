# Pull Up Method

## Problem

Your subclasses have methods that perform similar work.

## Solution

Make the methods identical and then move them to the relevant superclass.

## How to Refactor

1. Investigate similar methods in superclasses. If they aren’t identical, format them to match each other.

2. If methods use a different set of parameters, put the parameters in the form that you want to see in the superclass.

3. Copy the method to the superclass. Here you may find that the method code uses fields and methods that exist only in subclasses and therefore aren’t available in the superclass. To solve this, you can:

   - For fields: use either **Pull Up Field** or Self-**Encapsulate Field** to create getters and setters in subclasses; then declare these getters abstractly in the superclass.

   - For methods: use either **Pull Up Method** or declare abstract methods for them in the superclass (note that your class will become abstract if it wasn’t previously).

4. Remove the methods from the subclasses.

5. Check the locations in which the method is called. In some places you may be able to replace use of a subclass with the superclass.

### Anti-refactoring

- Push Down Method

### Similar refactorings

- Pull Up Field

### Helps other refactorings

- Form Template Method

### Eliminates smell

- Duplicate Code
