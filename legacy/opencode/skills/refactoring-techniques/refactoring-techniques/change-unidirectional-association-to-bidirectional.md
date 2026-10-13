# Change Unidirectional Association to Bidirectional

## Problem

You have two classes that each need to use the features of the other, but the association between them is only unidirectional.

## Solution

Add the missing association to the class that needs it.

## How to Refactor

1. Add a field for holding the reverse association.

2. Decide which class will be “dominant”. This class will contain the methods that create or update the association as elements are added or changed, establishing the association in its class and calling the utility methods for establishing the association in the associated object.

3. Create a utility method for establishing the association in the “non-dominant” class. The method should use what it’s given in parameters to complete the field. Give the method an obvious name so that it isn’t used later for any other purposes.

4. If old methods for controlling the unidirectional association were in the “dominant” class, complement them with calls to utility methods from the associated object.

5. If the old methods for controlling the association were in the “non-dominant” class, create the methods in the “dominant” class, call them, and delegate execution to them.

## Anti-refactoring

- Change Bidirectional Association to Unidirectional
