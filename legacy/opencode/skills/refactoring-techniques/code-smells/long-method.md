# Long Method

## Signs

A method contains too many lines of code. Generally, any method longer than ten lines should make you start asking questions.

## Treatment

As a rule of thumb, if you feel the need to comment on something inside a method, you should take this code and put it in a new method. Even a single line can and should be split off into a separate method, if it requires explanations. And if the method has a descriptive name, nobody will need to look at the code to see what it does.

- To reduce the length of a method body, use **Extract Method**.

- If local variables and parameters interfere with extracting a method, use **Replace Temp with Query**, **Introduce Parameter Object** or **Preserve Whole Object**.

- If none of the previous recipes help, try moving the entire method to a separate object via **Replace Method with Method Object**.

- Conditional operators and loops are a good clue that code can be moved to a separate method. For conditionals, use **Decompose Conditional**. If loops are in the way, try **Extract Method**.
