# Change Value to Reference

## Problem

So you have many identical instances of a single class that you need to replace with a single object.

## Solution

Convert the identical objects to a single reference object.

## How to Refactor

1. Use **Replace Constructor with Factory Method** on the class from which the references are to be generated.

2. Determine which object will be responsible for providing access to references. Instead of creating a new object, when you need one you now need to get it from a storage object or static dictionary field.

3. Determine whether references will be created in advance or dynamically as necessary. If objects are created in advance, make sure to load them before use.

4. Change the factory method so that it returns a reference. If objects are created in advance, decide how to handle errors when a non-existent object is requested. You may also need to use **Rename Method** to inform that the method returns only existing objects.

### Anti-refactoring

- Change Reference to Value
