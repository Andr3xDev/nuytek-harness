# Inline Class

## Problem

A class does almost nothing and isn’t responsible for anything, and no additional responsibilities are planned for it.

## Solution

Move all features from the class to another one.

## How to Refactor

1. In the recipient class, create the public fields and methods present in the donor class. Methods should refer to the equivalent methods of the donor class.
2. Replace all references to the donor class with references to the fields and methods of the recipient class.
3. Now test the program and make sure that no errors have been added. If tests show that everything is working A-OK, start using Move Method and Move Field to completely transplant all functionality to the recipient class from the original one. Continue doing so until the original class is completely empty.
4. Delete the original class.

### Anti-refactoring

- Extract Class

### Eliminates smell

- Shotgun Surgery
- Lazy Class
- Speculative Generality
