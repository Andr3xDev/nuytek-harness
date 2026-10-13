# Encapsulate Collection

## Problem

A class contains a collection field and a simple getter and setter for working with the collection.

## Solution

Make the getter-returned value read-only and create methods for adding/deleting elements of the collection.

## How to Refactor

1. Create methods for adding and deleting collection elements. They must accept collection elements in their parameters.

2. Assign an empty collection to the field as the initial value if this isn’t done in the class constructor.

3. Find the calls of the collection field setter. Change the setter so that it uses operations for adding and deleting elements, or make these operations call client code.

   Note that setters can be used only to replace all collection elements with other ones. Therefore it may be advisable to change the setter name (**Rename Method**) to `replace`.

4. Find all calls of the collection getter after which the collection is changed. Change the code so that it uses your new methods for adding and deleting elements from the collection.

5. Change the getter so that it returns a read-only representation of the collection.

6. Inspect the client code that uses the collection for code that would look better inside of the collection class itself.

### Eliminates smell

- Data Class
