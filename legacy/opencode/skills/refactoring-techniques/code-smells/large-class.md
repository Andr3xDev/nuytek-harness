# Large Class

## Signs

A class contains many fields/methods/lines of code.

## Treatment

When a class is wearing too many (functional) hats, think about splitting it up:

- **Extract Class** helps if part of the behavior of the large class can be spun off into a separate component.

- **Extract Subclass** helps if part of the behavior of the large class can be implemented in different ways or is used in rare cases.

- **Extract Interface** helps if it’s necessary to have a list of the operations and behaviors that the client can use.

- If a large class is responsible for the graphical interface, you may try to move some of its data and behavior to a separate domain object. In doing so, it may be necessary to store copies of some data in two places and keep the data consistent. **Duplicate Observed Data** offers a way to do this.
