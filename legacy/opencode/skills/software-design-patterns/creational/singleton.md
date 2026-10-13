# Singleton

## Structure

1. The **Singleton** class declares the static method `get_instance` that returns the same instance of its own class.

The Singleton’s constructor should be hidden from the client code. Calling the `get_instance` method should be the only way of getting the Singleton object.

## Relations with Other Patterns

- A **Facade** class can often be transformed into a **Singleton** since a single facade object is sufficient in most cases.

- Flyweight would resemble **Singleton** if you somehow managed to reduce all shared states of the objects to just one flyweight object. But there are two fundamental differences between these patterns:

   1. There should be only one **Singleton** instance, whereas a **Flyweight** class can have multiple instances with different intrinsic states.
   2. The **Singleton** object can be mutable. Flyweight objects are immutable.

- **Abstract Factories**, **Builders** and **Prototypes** can all be implemented as **Singletons**.
