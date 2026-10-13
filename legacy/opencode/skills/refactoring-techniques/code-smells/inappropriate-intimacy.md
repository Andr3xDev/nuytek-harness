# Inappropriate Intimacy

## Signs

One class uses the internal fields and methods of another class.

## Treatment

- The simplest solution is to use **Move Method** and **Move Field** to move parts of one class to the class in which those parts are used. But this works only if the first class truly doesn’t need these parts.

- Another solution is to use **Extract Class** and **Hide Delegate** on the class to make the code relations “official”.

- If the classes are mutually interdependent, you should use **Change Bidirectional Association to Unidirectional**.

- If this “intimacy” is between a subclass and the superclass, consider **Replace Delegation with Inheritance**.
