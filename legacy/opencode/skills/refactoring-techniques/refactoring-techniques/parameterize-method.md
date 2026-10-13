# Parameterize Method

## Problem

Multiple methods perform similar actions that are different only in their internal values, numbers or operations.

## Solution

Combine these methods by using a parameter that will pass the necessary special value.

## How to Refactor

1. Create a new method with a parameter and move it to the code that’s the same for all classes, by applying **Extract Method**. Note that sometimes only a certain part of methods is actually the same. In this case, refactoring consists of extracting only the same part to a new method.

2. In the code of the new method, replace the special/differing value with a parameter.

3. For each old method, find the places where it’s called, replacing these calls with calls to the new method that include a parameter. Then delete the old method.

### Anti-refactoring

- Replace Parameter with Explicit Methods

### Similar refactorings

- Extract Method
- Form Template Method

### Eliminates smell

- Duplicate Code
