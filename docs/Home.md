# amalgomate

**amalgomate** is a Go program that programmatically combines the source code of separate stand-alone Go programs into a
single combined program. It also provides a library and invocation mechanism for invoking the original programs from the
combined program.

amalgomate can be used to create a single executable that combines the functionality of several stand-alone executables.
This provides advantages such as reducing the over-all size, locking in versions, creating a single point of execution
and allowing the program to implement logic that makes use of its component programs.
