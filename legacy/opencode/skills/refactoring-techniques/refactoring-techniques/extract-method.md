# Extract Method

## Problem

You have a code fragment that can be grouped together.

## Solution

Move this code to a separate new method (or function) and replace the old code with a call to the method.

## How to Refactor

1. Create a new method and name it in a way that makes its purpose self-evident.

2. Copy the relevant code fragment to your new method. Delete the fragment from its old location and put a call for the new method there instead. Find all variables used in this code fragment. If they’re declared inside the fragment and not used outside of it, simply leave them unchanged—they’ll become local variables for the new method.

3. If the variables are declared prior to the code that you’re extracting, you will need to pass these variables to the parameters of your new method in order to use the values previously contained in them. Sometimes it’s easier to get rid of these variables by resorting to Replace Temp with Query.

4. If you see that a local variable changes in your extracted code in some way, this may mean that this changed value will be needed later in your main method. Double-check! And if this is indeed the case, return the value of this variable to the main method to keep everything functioning.

### Anti-refactoring

- Inline Method

### Similar Refactorings

- Move Method

### Helps other refactorings

- Introduce Parameter Object
- Form Template Method
- Parameterize Method

### Eliminates smell

- Duplicate Code
- Long Method
- Feature Envy
- Switch Statements
- Message Chains
- Comments
- Data Class
