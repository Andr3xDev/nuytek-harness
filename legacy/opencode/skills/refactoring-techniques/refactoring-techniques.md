# Refactoring Techniques

## Composing Methods

Much of refactoring is devoted to correctly composing methods. In most cases, excessively long methods are the root of all evil. The vagaries of code inside these methods conceal the execution logic and make the method extremely hard to understand—and even harder to change.

The refactoring techniques in this group streamline methods, remove code duplication, and pave the way for future improvements.

- [Extract Method](./refactoring-techniques/extract-method.md)
- [Inline Method](./refactoring-techniques/inline-method.md)
- [Extract Variable](./refactoring-techniques/extract-variable.md)
- [Inline Temp](./refactoring-techniques/inline-temp.md)
- [Replace Temp with Query](./refactoring-techniques/replace-temp-with-query.md)
- [Split Temporary Variable](./refactoring-techniques/split-temporary-variable.md)
- [Remove Assignments to Parameters](./refactoring-techniques/remove-assignments-to-parameters.md)
- [Replace Method with Method Object](./refactoring-techniques/replace-method-with-method-object.md)
- [Substitute Algorithm](./refactoring-techniques/substitute-algorithm.md)

## Moving Features Between Objects

Even if you have distributed functionality among different classes in a less-than-perfect way, there is still hope.

These refactoring techniques show how to safely move functionality between classes, create new classes, and hide implementation details from public access.

- [Move Method](./refactoring-techniques/move-method.md)
- [Move Field](./refactoring-techniques/move-field.md)
- [Extract Class](./refactoring-techniques/extract-class.md)
- [Inline Class](./refactoring-techniques/inline-class.md)
- [Hide Delegate](./refactoring-techniques/hide-delegate.md)
- [Remove Middle Man](./refactoring-techniques/remove-middle-man.md)
- [Introduce Foreign Method](./refactoring-techniques/introduce-foreign-method.md)
- [Introduce Local Extension](./refactoring-techniques/introduce-local-extension.md)

## Organizing Data

These refactoring techniques help with data handling, replacing primitives with rich class functionality. Another important result is untangling of class associations, which makes classes more portable and reusable.

- [Change Value to Reference](./refactoring-techniques/change-value-to-reference.md)
- [Change Reference to Value](./refactoring-techniques/change-reference-to-value.md)
- [Duplicate Observed Data](./refactoring-techniques/duplicate-observed-data.md)
- [Self Encapsulate Field](./refactoring-techniques/self-encapsulate-field.md)
- [Replace Data Value with Object](./refactoring-techniques/replace-data-value-with-object.md)
- [Replace Array with Object](./refactoring-techniques/replace-array-with-object.md)
- [Change Unidirectional Association to Bidirectional](./refactoring-techniques/change-unidirectional-association-to-bidirectional.md)
- [Change Bidirectional Association to Unidirectional](./refactoring-techniques/change-bidirectional-association-to-unidirectional.md)
- [Encapsulate Field](./refactoring-techniques/encapsulate-field.md)
- [Encapsulate Collection](./refactoring-techniques/encapsulate-collection.md)
- [Replace Magic Number with Symbolic Constant](./refactoring-techniques/replace-magic-number-with-symbolic-constant.md)
- [Replace Type Code with Class](./refactoring-techniques/replace-type-code-with-class.md)
- [Replace Type Code with Subclasses](./refactoring-techniques/replace-type-code-with-subclasses.md)
- [Replace Type Code with State/Strategy](./refactoring-techniques/replace-type-code-with-state-strategy.md)
- [Replace Subclass with Fields](./refactoring-techniques/replace-subclass-with-fields.md)

## Simplifying Conditional Expressions

Conditionals tend to get more and more complicated in their logic over time, and there are yet more techniques to combat this as well.

- [Consolidate Conditional Expression](./refactoring-techniques/consolidate-conditional-expression.md)
- [Consolidate Duplicate Conditional Fragments](./refactoring-techniques/consolidate-duplicate-conditional-fragments.md)
- [Decompose Conditional](./refactoring-techniques/decompose-conditional.md)
- [Replace Conditional with Polymorphism](./refactoring-techniques/replace-conditional-with-polymorphism.md)
- [Replace Control Flag](./refactoring-techniques/replace-control-flag.md)
- [Replace Nested Conditional with Guard Clauses](./refactoring-techniques/replace-nested-conditional-with-guard-clauses.md)
- [Introduce Null Object](./refactoring-techniques/introduce-null-object.md)
- [Introduce Assertion](./refactoring-techniques/introduce-assertion.md)

## Simplifying Methods Calls

These techniques make method calls simpler and easier to understand. This, in turn, simplifies the interfaces for interaction between classes.

- [Add Parameter](./refactoring-techniques/add-parameter.md)
- [Remove Parameter](./refactoring-techniques/remove-parameter.md)
- [Rename Method](./refactoring-techniques/rename-method.md)
- [Separate Query from Modifier](./refactoring-techniques/separate-query-from-modifier.md)
- [Parameterize Method](./refactoring-techniques/parameterize-method.md)
- [Introduce Parameter Object](./refactoring-techniques/introduce-parameter-object.md)
- [Preserve Whole Object](./refactoring-techniques/preserve-whole-object.md)
- [Remove Setting Method](./refactoring-techniques/remove-setting-method.md)
- [Replace Parameter with Explicit Methods](./refactoring-techniques/replace-parameter-with-explicit-methods.md)
- [Replace Parameter with Method Call](./refactoring-techniques/replace-parameter-with-method-call.md)
- [Hide Method](./refactoring-techniques/hide-method.md)
- [Replace Constructor with Factory Method](./refactoring-techniques/replace-constructor-with-factory-method.md)
- [Replace Error Code with Exception](./refactoring-techniques/replace-error-code-with-exception.md)
- [Replace Exception with Test](./refactoring-techniques/replace-exception-with-test.md)

## Dealing with Generalization

Abstraction has its own group of refactoring techniques, primarily associated with moving functionality along the class inheritance hierarchy, creating new classes and interfaces, and replacing inheritance with delegation and vice versa.

- [Pull Up Field](./refactoring-techniques/pull-up-field.md)
- [Pull Up Method](./refactoring-techniques/pull-up-method.md)
- [Pull Up Constructor Body](./refactoring-techniques/pull-up-constructor-body.md)
- [Pull Down Field](./refactoring-techniques/pull-down-field.md)
- [Pull Down Method](./refactoring-techniques/pull-down-method.md)
- [Extract Subclass](./refactoring-techniques/extract-subclass.md)
- [Extract Superclass](./refactoring-techniques/extract-superclass.md)
- [Extract Interface](./refactoring-techniques/extract-interface.md)
- [Collapse Hierarchy](./refactoring-techniques/collapse-hierarchy.md)
- [Form Template Method](./refactoring-techniques/form-template-method.md)
- [Replace Inheritance with Delegation](./refactoring-techniques/replace-inheritance-with-delegation.md)
- [Replace Delegation with Inheritance](./refactoring-techniques/replace-delegation-with-inheritance.md)
