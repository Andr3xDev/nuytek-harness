# Replace Magic Number with Symbolic Constant

## Problem

Your code uses a number that has a certain meaning to it.

## Solution

Replace this number with a constant that has a human-readable name explaining the meaning of the number.

## How to Refactor

1. Declare a constant and assign the value of the magic number to it.

2. Find all mentions of the magic number.

3. For each of the numbers that you find, double-check that the magic number in this particular case corresponds to the purpose of the constant. If yes, replace the number with your constant. This is an important step, since the same number can mean absolutely different things (and replaced with different constants, as the case may be).
