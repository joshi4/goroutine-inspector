# Goroutine Inspector

## Motivation

Often I found myself in the following scenario: TestXXX byitself passes but running the whole test suite results in an error.

Usually, this means theres an unexpected mutation of global state caused by the interaction between two or more test cases.

A large fraction of the time the root cause of the problem is that there is a go routine leak somewhere.

From here we have to audit the code carefully or resort to adding debug print statements to figure out which goroutine has escaped.

Goroutine Inspector makes the task of finding leaked goroutines as easy as adding a couple of lines of code to the test suite. It also ensures that any go routines that leak are caught immediately, thus ensuring a clean and safe codebase.

## Example Usage

## Caveat

Under the hood goroutine-inspector relies on an execution trace as its source of truth, if due to the vagaries of the scheduler a goroutine doesn't emit the `GoroutineStop` event it will be considered as having leaked, so some false positives are possible.

## Contribution

PR's and bug reports are welcome :)
