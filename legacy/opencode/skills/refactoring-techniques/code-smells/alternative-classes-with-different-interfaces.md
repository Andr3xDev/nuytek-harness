# Alternative Classes with Different Interfaces

## Signs

Two classes perform identical functions but have different method names.

## Treatment

Try to put the interface of classes in terms of a common denominator:

- **Rename Method** to make them identical in all alternative classes.
- **Move Method**, **Add Parameter** and **Parameterize Method** to make the signature and implementation of methods the same.
- If only part of the functionality of the classes is duplicated, try using **Extract Superclass**. In this case, the existing classes will become subclasses.
- After you have determined which treatment method to use and implemented it, you may be able to delete one of the classes.
