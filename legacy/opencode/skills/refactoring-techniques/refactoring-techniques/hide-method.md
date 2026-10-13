# Hide Method

## Problem

A method isn’t used by other classes or is used only inside its own class hierarchy.

## Solution

Make the method private or protected.

## How to Refactor

1. Regularly try to find methods that can be made private. Static code analysis and good unit test coverage can offer a big leg up.

2. Make each method as private as possible.

### Eliminates smell

- Data Class
