# Hide Delegate

## Problem

The client gets object B from a field or method of object А. Then the client calls a method of object B.

## Solution

Create a new method in class A that delegates the call to object B. Now the client doesn’t know about, or depend on, class B.

## How to Refactor

1. For each method of the delegate-class called by the client, create a method in the server-class that delegates the call to the delegate-class.
2. Change the client code so that it calls the methods of the server-class.
3. If your changes free the client from needing the delegate-class, you can remove the access method to the delegate-class from the server-class (the method that was originally used to get the delegate-class).

### Anti-refactoring

- Remove Middle Man

### Eliminates smell

- Message Chains
- Inappropriate Intimacy
