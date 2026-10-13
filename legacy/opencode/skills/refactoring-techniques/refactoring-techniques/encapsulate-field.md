# Encapsulate Field

## Problem

You have a public field.

## Solution

Make the field private and create access methods for it.

## How to Refactor

1. Create a getter and setter for the field.

2. Find all invocations of the field. Replace receipt of the field value with the getter, and replace setting of new field values with the setter.

3. After all field invocations have been replaced, make the field private.

## Next Steps

Encapsulate Field is only the first step in bringing data and the behaviors involving this data closer together. After you create simple methods for access fields, you should recheck the places where these methods are called. It’s quite possible that the code in these areas would look more appropriate in the access methods.

### Similar refactorings

- Self Encapsulate Field: Create getters and setters for a field instead of direct access within the class' methods.

### Eliminates smell

- Data Class
