/*
Package runtime provides utilities for Go's runtime system in addition to the
standard `runtime` package.

For now, the package provides utility to convert the CPU quota (applied through
Linux kernel's CFS process scheduler) of the current process to a proper value
for `GOMAXPROCS`.
*/
package runtime
