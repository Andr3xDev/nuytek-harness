# Duplicate Code

## Signs

Two code fragments look almost identical.

## Treatment

- If the same code is found in two subclasses of the same level:

  - Use **Extract Method** for both classes, followed by **Pull Up Field** for the fields used in the method that you’re pulling up.

  - If the duplicate code is inside a constructor, use **Pull Up Constructor Body**.

  - If the duplicate code is similar but not completely identical, use **Form Template Method**.

  - If two methods do the same thing but use different algorithms, select the best algorithm and apply **Substitute Algorithm**.

- If duplicate code is found in two different classes:

  - If the classes aren’t part of a hierarchy, use **Extract Superclass** in order to create a single superclass for these classes that maintains all the previous functionality.

  - If it’s difficult or impossible to create a superclass, use **Extract Class** in one class and use the new component in the other.

- If a large number of conditional expressions are present and perform the same code (differing only in their conditions), merge these operators into a single condition using **Consolidate Conditional Expression** and use **Extract Method** to place the condition in a separate method with an easy-to-understand name.

- If the same code is performed in all branches of a conditional expression: place the identical code outside of the condition tree by using **Consolidate Duplicate Conditional Fragments**.
