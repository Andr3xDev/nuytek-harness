# Extract Class

## Problem

When one class does the work of two, awkwardness results.

## Solution

Instead, create a new class and place the fields and methods responsible for the relevant functionality in it.

## How to Refactor

Before starting, decide on how exactly you want to split up the responsibilities of the class.

1. Create a new class to contain the relevant functionality.
2. Create a relationship between the old class and the new one. Optimally, this relationship is unidirectional; this allows reusing the second class without any issues. Nonetheless, if you think that a two-way relationship is necessary, this can always be set up.
3. Use Move Field and Move Method for each field and method that you have decided to move to the new class. For methods, start with private ones in order to reduce the risk of making a large number of errors. Try to relocate just a little bit at a time and test the results after each move, in order to avoid a pileup of error-fixing at the very end. After you’re done moving, take one more look at the resulting classes. An old class with changed responsibilities may be renamed for increased clarity. Check again to see whether you can get rid of two-way class relationships, if any are present.
4. Also give thought to accessibility to the new class from the outside. You can hide the class from the client entirely by making it private, managing it via the fields from the old class. Alternatively, you can make it a public one by allowing the client to change values directly. Your decision here depends on how safe it’s for the behavior of the old class when unexpected direct changes are made to the values in the new class.

### Anti-refactoring

- Inline Class

### Similar refactorings

- Extract Subclass
- Replace Data Value with Object

### Eliminates smell

- Duplicate Code
- Large Class
- Divergent Change
- Data Clumps
- Primitive Obsession
- Temporary Field
- Inappropriate Intimacy
