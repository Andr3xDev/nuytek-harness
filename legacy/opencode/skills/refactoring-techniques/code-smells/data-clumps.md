# Data Clumps

## Signs

Sometimes different parts of the code contain identical groups of variables (such as parameters for connecting to a database). These clumps should be turned into their own classes.

## Treatment

- If repeating data comprises the fields of a class, use **Extract Class** to move the fields to their own class.

- If the same data clumps are passed in the parameters of methods, use **Introduce Parameter Object** to set them off as a class.

- If some of the data is passed to other methods, think about passing the entire data object to the method instead of just individual fields. **Preserve Whole Object** will help with this.

- Look at the code used by these fields. It may be a good idea to move this code to a data class.
