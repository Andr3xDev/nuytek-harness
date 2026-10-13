# Decompose Conditional

## Problem

You have a complex conditional (if-then/else or switch).

## Solution

Decompose the complicated parts of the conditional into separate methods: the condition, then and else.

## How to Refactor

1. Extract the conditional to a separate method via **Extract Method**.

2. Repeat the process for the `then` and `else` blocks.

### Eliminates smell

- Long Method
