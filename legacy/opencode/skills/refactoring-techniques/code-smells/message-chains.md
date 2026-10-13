# Message Chains

## Signs

In code you see a series of calls resembling $a->b()->c()->d()

## Treatment

- To delete a message chain, use **Hide Delegate**.

- Sometimes it’s better to think of why the end object is being used. Perhaps it would make sense to use **Extract Method** for this functionality and move it to the beginning of the chain, by using **Move Method**.
