# Divergent Change

## Signs

You find yourself having to change many unrelated methods when you make changes to a class. For example, when adding a new product type you have to change the methods for finding, displaying, and ordering products.

## Treatment

- Split up the behavior of the class via **Extract Class**.

- If different classes have the same behavior, you may want to combine the classes through inheritance (**Extract Superclass** and **Extract Subclass**).
