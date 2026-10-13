# Temporary Field

## Signs

Temporary fields get their values (and thus are needed by objects) only under certain circumstances. Outside of these circumstances, they’re empty.

## Treatment

- Temporary fields and all code operating on them can be put in a separate class via **Extract Class**. In other words, you’re creating a method object, achieving the same result as if you would perform **Replace Method with Method Object**.

- **Introduce Null Object** and integrate it in place of the conditional code which was used to check the temporary field values for existence.
