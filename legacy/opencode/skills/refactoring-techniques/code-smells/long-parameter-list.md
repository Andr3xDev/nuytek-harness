# Long Parameter List

## Signs and Symptoms

More than three or four parameters for a method.

## Treatment

- Check what values are passed to parameters. If some of the arguments are just results of method calls of another object, use **Replace Parameter with Method Call**. This object can be placed in the field of its own class or passed as a method parameter.

- Instead of passing a group of data received from another object as parameters, pass the object itself to the method, by using **Preserve Whole Object**.

- But if these parameters are coming from different sources, you can pass them as a single parameter object via **Introduce Parameter Object**.
