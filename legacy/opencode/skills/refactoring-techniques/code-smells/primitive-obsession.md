# Primitive Obsession

## Signs

- Use of primitives instead of small objects for simple tasks (such as currency, ranges, special strings for phone numbers, etc.)

- Use of constants for coding information (such as a constant `USER_ADMIN_ROLE = 1` for referring to users with administrator rights.)

- Use of string constants as field names for use in data arrays.

## Treatment

- If you have a large variety of primitive fields, it may be possible to logically group some of them into their own class. Even better, move the behavior associated with this data into the class too. For this task, try **Replace Data Value with Object**.

- If the values of primitive fields are used in method parameters, go with **Introduce Parameter Object** or **Preserve Whole Object**.

- When complicated data is coded in variables, use **Replace Type Code with Class**, **Replace Type Code with Subclasses** or **Replace Type Code with State/Strategy**.

- If there are arrays among the variables, use **Replace Array with Object**.
