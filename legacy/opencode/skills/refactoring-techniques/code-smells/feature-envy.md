# Feature Envy

## Signs

A method accesses the data of another object more than its own data.

## Treatment

As a basic rule, if things change at the same time, you should keep them in the same place. Usually data and functions that use this data are changed together (although exceptions are possible).

- If a method clearly should be moved to another place, use **Move Method**.

- If only part of a method accesses the data of another object, use **Extract Method** to move the part in question.

- If a method uses functions from several other classes, first determine which class contains most of the data used. Then place the method in this class along with the other data. Alternatively, use **Extract Method** to split the method into several parts that can be placed in different places in different classes.
