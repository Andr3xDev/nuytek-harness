# Replace Parameter with Method Call

## Problem

Calling a query method and passing its results as the parameters of another method, while that method could call the query directly.

## Solution

Instead of passing the value through a parameter, try placing a query call inside the method body.

## How to Refactor

1. Make sure that the value-getting code doesn’t use parameters from the current method, since they’ll be unavailable from inside another method. If so, moving the code isn’t possible.

2. If the relevant code is more complicated than a single method or function call, use **Extract Method** to isolate this code in a new method and make the call simple.

3. In the code of the main method, replace all references to the parameter being replaced with calls to the method that gets the value.

4. Use **Remove Parameter** to eliminate the now-unused parameter.
