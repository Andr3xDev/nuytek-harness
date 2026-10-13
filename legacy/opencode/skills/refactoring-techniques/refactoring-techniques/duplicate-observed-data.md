# Duplicate Observed Data

## Problem

Is domain data stored in classes responsible for the GUI?

## Solution

Then it’s a good idea to separate the data into separate classes, ensuring connection and synchronization between the domain class and the GUI.

## How to Refactor

1. Hide direct access to domain data in the GUI class. For this, it’s best to use Self Encapsulate Field. So you create the getters and setters for this data.

2. In handlers for GUI class events, use setters to set new field values. This will let you pass these values to the associated domain object.

3. Create a domain class and copy necessary fields from the GUI class to it. Create getters and setters for all these fields.

4. Create an Observer pattern for these two classes:

   - In the domain class, create an array for storing observer objects (GUI objects), as well as methods for registering, deleting and notifying them.

   - In the GUI class, create a field for storing references to the domain class as well as the update() method, which will be reacting to changes in the object and update the values of fields in the GUI class. Note that value updates should be established directly in the method, in order to avoid recursion.

   - In the GUI class constructor, create an instance of domain class and save it in the field you have created. Register the GUI object as an observer in the domain object.

   - In the setters for domain class fields, call the method for notifying the observer (in other words, method for updating in the GUI class), in order to pass the new values to the GUI.

   - Change the setters of the GUI class fields so that they set new values in the domain object directly. Watch out to make sure that values aren’t set through a domain class setter—otherwise infinite recursion will result.

### Implements design pattern

- Observer

### Eliminates smell

- Large Class
