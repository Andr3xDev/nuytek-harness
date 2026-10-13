# Comments

## Signs

A method is filled with explanatory comments. If you feel that a code fragment can’t be understood without comments, try to change the code structure in a way that makes comments unnecessary.

## Treatment

- If a comment is intended to explain a complex expression, the expression should be split into understandable subexpressions using **Extract Variable**

- If a comment explains a section of code, this section can be turned into a separate method via **Extract Method**. The name of the new method can be taken from the comment text itself, most likely.

- If a method has already been extracted, but comments are still necessary to explain what the method does, give the method a self-explanatory name. Use **Rename Method** for this.

- If you need to assert rules about a state that’s necessary for the system to work, use **Introduce Assertion**.
