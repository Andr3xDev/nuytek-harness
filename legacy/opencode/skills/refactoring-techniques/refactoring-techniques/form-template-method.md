# Form Template Method

## Problem

Your subclasses implement algorithms that contain similar steps in the same order.

## Solution

Move the algorithm structure and identical steps to a superclass, and leave implementation of the different steps in the subclasses.

## How to Refactor

1. Split algorithms in the subclasses into their constituent parts described in separate methods. **Extract Method** can help with this.

2. The resulting methods that are identical for all subclasses can be moved to a superclass via **Pull Up Method**.

3. The non-similar methods can be given consistent names via **Rename Method**.

4. Move the signatures of non-similar methods to a superclass as abstract ones by using **Pull Up Method**. Leave their implementations in the subclasses.

5. And finally, pull up the main method of the algorithm to the superclass. Now it should work with the method steps described in the superclass, both real and abstract.

### Implements design pattern

- Template Method

### Eliminates smell

- Duplicate Code
