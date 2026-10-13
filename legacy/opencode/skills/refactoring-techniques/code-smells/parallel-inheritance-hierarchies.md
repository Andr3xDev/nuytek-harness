# Parallel Inheritance Hierarchies

## Signs

Whenever you create a subclass for a class, you find yourself needing to create a subclass for another class.

## Treatment

You may de-duplicate parallel class hierarchies in two steps. First, make instances of one hierarchy refer to instances of another hierarchy. Then, remove the hierarchy in the referred class, by using **Move Method** and **Move Field**.
