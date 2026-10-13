# Extract Subclass

## Problem

A class has features that are used only in certain cases.

## Solution

Create a subclass and use it in these cases.

## How to Refactor

1. Create a new subclass from the class of interest.

2. If you need additional data to create objects from a subclass, create a constructor and add the necessary parameters to it. Don’t forget to call the constructor’s parent implementation.

3. Find all calls to the constructor of the parent class. When the functionality of a subclass is necessary, replace the parent constructor with the subclass constructor.

4. Move the necessary methods and fields from the parent class to the subclass. Do this via **Push Down Method** and **Push Down Field**. It’s simpler to start by moving the methods first. This way, the fields remain accessible throughout the whole process: from the parent class prior to the move, and from the subclass itself after the move is complete.

5. After the subclass is ready, find all the old fields that controlled the choice of functionality. Delete these fields by using polymorphism to replace all the operators in which the fields had been used. A simple example: in the Car class, you had the field `is_electric_car` and, depending on it, in the `refuel()` method the car is either fueled up with gas or charged with electricity. Post-refactoring, the `is_electric_car` field is removed and the Car and `ElectricCar` classes will have their own implementations of the `refuel()` method.

### Similar refactorings

- Extract Class

### Eliminates smell

- Large Class
