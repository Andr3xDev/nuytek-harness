# Substitute Algorithm

## Problem

So you want to replace an existing algorithm with a new one?

## Solution

Replace the body of the method that implements the algorithm with a new algorithm.

## How to Refactor

1. Make sure that you have simplified the existing algorithm as much as possible. Move unimportant code to other methods using Extract Method. The fewer moving parts in your algorithm, the easier it’s to replace.
2. Create your new algorithm in a new method. Replace the old algorithm with the new one and start testing the program.
3. If the results don’t match, return to the old implementation and compare the results. Identify the causes of the discrepancy. While the cause is often an error in the old algorithm, it’s more likely due to something not working in the new one.
4. When all tests are successfully completed, delete the old algorithm for good!

### Eliminates smell

- Duplicate Code
- Long Method
