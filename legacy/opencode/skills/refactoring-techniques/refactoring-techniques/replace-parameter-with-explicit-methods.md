# Replace Parameter with Explicit Methods

## Problem

A method is split into parts, each of which is run depending on the value of a parameter.

## Solution

Extract the individual parts of the method into their own methods and call them instead of the original method.

## When Not to Use

Don’t replace a parameter with explicit methods if a method is rarely changed and new variants aren’t added inside it.

## How to Refactor

1. For each variant of the method, create a separate method. Run these methods based on the value of a parameter in the main method.

2. Find all places where the original method is called. In these places, place a call for one of the new parameter-dependent variants.

3. When no calls to the original method remain, delete it.

### Anti-refactoring

- Parameterize Method

### Similar refactorings

- Replace Conditional with Polymorphism

### Eliminates smell

- Switch Statements
- Long Method
